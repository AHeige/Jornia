package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type JournalConfig struct {
	Root           string `json:"root"`
	Enabled        bool   `json:"enabled"`
	Revision       int    `json:"revision"`
	ConsentVersion int    `json:"consent_version"`
}
type JournalStatus struct {
	Config     JournalConfig `json:"config"`
	Scanning   bool          `json:"scanning"`
	Files      int           `json:"files"`
	Characters int           `json:"characters"`
	LastScan   int64         `json:"last_scan"`
	LastImport int64         `json:"last_import"`
	Error      string        `json:"error"`
}
type journalStamp struct{ Size, Modified int64 }
type JournalSync struct {
	store  *Store
	mu     sync.Mutex
	status JournalStatus
	seen   map[string]string
	stamps map[string]journalStamp
	wake   chan struct{}
}

func NewJournalSync(s *Store) (*JournalSync, error) {
	j := &JournalSync{store: s, seen: map[string]string{}, stamps: map[string]journalStamp{}, wake: make(chan struct{}, 1)}
	var raw string
	err := s.DB.QueryRow(`SELECT value FROM local_settings WHERE key='journal'`).Scan(&raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &j.status.Config)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return j, err
}
func (j *JournalSync) Status() JournalStatus { j.mu.Lock(); defer j.mu.Unlock(); return j.status }
func (j *JournalSync) Run(ctx context.Context) {
	j.Scan(ctx)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.Scan(ctx)
		case <-j.wake:
			j.Scan(ctx)
		}
	}
}

type JournalIntent struct {
	ID               string `json:"mutation_id"`
	Root             string `json:"root"`
	Enabled          bool   `json:"enabled"`
	ConsentVersion   int    `json:"consent_version"`
	ExpectedRevision int    `json:"expected_revision"`
}

func (j *JournalSync) Configure(ctx context.Context, req JournalIntent) (Receipt, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	fresh, err := j.store.begin(ctx, req.ID, "journal", "", Fingerprint(req))
	if err != nil {
		return Receipt{}, err
	}
	if !fresh {
		return j.store.Operation(ctx, req.ID)
	}
	fail := func(message string) (Receipt, error) {
		_, _ = j.store.DB.ExecContext(ctx, `UPDATE operations SET status='failed',error=? WHERE id=?`, message, req.ID)
		return Receipt{}, errors.New(message)
	}
	if req.ExpectedRevision != j.status.Config.Revision {
		return fail("Settings changed in another window. Reload saved state before changing them again.")
	}
	root := j.status.Config.Root
	if req.Enabled {
		if req.ConsentVersion != 1 && req.ConsentVersion != 2 {
			return fail("Approve local journal reading first.")
		}
		root = strings.TrimSpace(req.Root)
		if len(root) > 2048 || !filepath.IsAbs(root) || strings.HasPrefix(root, `\\`) {
			return fail("Enter the full local path to your WoW client folder.")
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			return fail("Could not open the WoW folder. Nothing was imported.")
		}
		info, err := os.Stat(filepath.Join(root, "WTF"))
		if err != nil || !info.IsDir() {
			return fail("Choose the client folder containing WTF.")
		}
	}
	config := JournalConfig{Root: root, Enabled: req.Enabled, Revision: j.status.Config.Revision + 1, ConsentVersion: req.ConsentVersion}
	raw, _ := json.Marshal(config)
	result, _ := json.Marshal(config)
	tx, err := j.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO local_settings(key,value) VALUES('journal',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(raw)); err != nil {
		return Receipt{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE operations SET status='completed',result=? WHERE id=?`, string(result), req.ID); err != nil {
		return Receipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return Receipt{}, err
	}
	j.status.Config = config
	j.status.Error = ""
	j.seen = map[string]string{}
	j.stamps = map[string]journalStamp{}
	select {
	case j.wake <- struct{}{}:
	default:
	}
	return j.store.Operation(ctx, req.ID)
}
func (j *JournalSync) Scan(ctx context.Context) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.status.Config.Enabled {
		return
	}
	j.status.Scanning = true
	defer func() { j.status.Scanning = false }()
	j.status.Error = ""
	j.status.Files = 0
	var discoveryErr error
	if j.status.Config.ConsentVersion >= 2 {
		j.status.Characters, discoveryErr = j.store.DiscoverCharacters(ctx, j.status.Config.Root)
	}
	if discoveryErr != nil && !errors.Is(discoveryErr, os.ErrNotExist) {
		j.status.Error = "Character directories could not be fully read."
	}
	root := filepath.Join(j.status.Config.Root, "WTF", "Account")
	entries := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			return walkErr
		}
		entries++
		if entries > 20000 {
			return errors.New("scan limit")
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() && strings.Count(rel, string(filepath.Separator)) > 5 {
			return fs.SkipDir
		}
		if d.IsDir() || d.Name() != "Wisp.lua" || filepath.Base(filepath.Dir(path)) != "SavedVariables" {
			return nil
		}
		// The five-component per-character file is the identity sidecar, not
		// the account-level event journal. Discovery reads it first (consent v2).
		if len(strings.Split(rel, string(filepath.Separator))) == 5 {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("journal symlink rejected")
		}
		j.status.Files++
		if j.status.Files > 100 {
			return errors.New("file limit")
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			file.Close()
			return errors.New("invalid journal file")
		}
		stamp := journalStamp{info.Size(), info.ModTime().UnixNano()}
		if j.stamps[path] == stamp {
			file.Close()
			return nil
		}
		b, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
		file.Close()
		if err != nil {
			return err
		}
		fingerprint := Fingerprint(string(b))
		if j.seen[path] == fingerprint {
			j.stamps[path] = stamp
			return nil
		}
		packs, err := ParseSavedVariables(string(b))
		if err != nil {
			return err
		}
		for _, pack := range packs {
			receipt, err := j.store.importData(ctx, "auto:"+Fingerprint(pack), pack, true)
			if err != nil {
				return err
			}
			if receipt.Status != "completed" {
				return errors.New("unconfirmed import")
			}
		}
		j.seen[path] = fingerprint
		j.stamps[path] = stamp
		j.status.LastImport = time.Now().Unix()
		return nil
	})
	j.status.LastScan = time.Now().Unix()
	if errors.Is(err, os.ErrNotExist) {
		j.status.Error = "No Jornia journal found yet. Enable the addon, log in and save with /reload or logout."
	} else if err != nil && ctx.Err() == nil {
		j.status.Error = "A Jornia journal could not be read. Existing history is kept; the file will be checked again."
	} else if j.status.Files == 0 {
		j.status.Error = "No Jornia journal found yet. Other addon contents and account settings are not read."
	}
}
