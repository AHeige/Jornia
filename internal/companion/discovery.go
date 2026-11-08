package companion

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Installation detection checks only conventional game locations. It does not read
// account/character directories before the player's persisted journal consent.
func DetectInstallations() []string {
	roots := []string{"/Applications/World of Warcraft", filepath.Join(os.Getenv("HOME"), "Applications", "World of Warcraft")}
	for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), `C:\Games`, `D:\Games`, `E:\Games`} {
		if base != "" {
			roots = append(roots, filepath.Join(base, "World of Warcraft"), filepath.Join(base, "WoW"))
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			roots = append(roots, dir)
			dir = filepath.Dir(dir)
		}
	}
	roots = append(roots, platformGameRoots()...)
	found := []string{}
	seen := map[string]bool{}
	for _, root := range roots {
		for _, flavor := range []string{"", "_classic_", "_classic_era_", "_classic_beta_", "_forever_"} {
			path := filepath.Join(root, flavor)
			info, err := os.Stat(filepath.Join(path, "WTF"))
			if err == nil && info.IsDir() {
				if resolved, err := filepath.EvalSymlinks(path); err == nil && !seen[resolved] {
					found = append(found, resolved)
					seen[resolved] = true
				}
			}
		}
	}
	return found
}
func (s *Store) initDiscovery() error {
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS character_files(folder_id TEXT PRIMARY KEY,character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE);
 CREATE TABLE IF NOT EXISTS journal_aliases(alias TEXT PRIMARY KEY,character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE);`)
	return err
}
func directoryEntries(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := []os.DirEntry{}
	for _, e := range entries {
		if e.IsDir() && e.Type()&os.ModeSymlink == 0 && e.Name() != "SavedVariables" && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e)
		}
	}
	if len(out) > 1000 {
		return nil, errors.New("character directory limit")
	}
	return out, nil
}

// DiscoverCharacters reads directory names and Jornia's own per-character identity
// marker. It never parses other addons, config caches or account credentials.
func (s *Store) DiscoverCharacters(ctx context.Context, root string) (int, error) {
	accounts, err := directoryEntries(filepath.Join(root, "WTF", "Account"))
	if err != nil {
		return 0, err
	}
	count := 0
	visited := len(accounts)
	for _, account := range accounts {
		realms, err := directoryEntries(filepath.Join(root, "WTF", "Account", account.Name()))
		if err != nil {
			return count, err
		}
		visited += len(realms)
		if visited > 20000 {
			return count, errors.New("directory discovery limit")
		}
		for _, realm := range realms {
			chars, err := directoryEntries(filepath.Join(root, "WTF", "Account", account.Name(), realm.Name()))
			if err != nil {
				return count, err
			}
			visited += len(chars)
			if visited > 20000 {
				return count, errors.New("directory discovery limit")
			}
			for _, char := range chars {
				if count >= 1000 {
					return count, errors.New("character discovery limit")
				}
				path := filepath.Join(root, "WTF", "Account", account.Name(), realm.Name(), char.Name())
				// WoW creates a per-character SavedVariables directory or config-cache.wtf.
				info, a := os.Lstat(filepath.Join(path, "SavedVariables"))
				cache, b := os.Lstat(filepath.Join(path, "config-cache.wtf"))
				if !(a == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0) && !(b == nil && cache.Mode().IsRegular()) {
					continue
				}
				folderID := "folder:" + Fingerprint(path)[:40]
				alias := ""
				if a == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
					var markerErr error
					alias, markerErr = readCharacterIdentity(filepath.Join(path, "SavedVariables", "Wisp.lua"))
					if markerErr != nil {
						return count, markerErr
					}
				}
				tx, err := s.DB.BeginTx(ctx, nil)
				if err != nil {
					return count, err
				}
				var excluded int
				err = tx.QueryRowContext(ctx, `SELECT count(*) FROM journey_exclusions WHERE character_id IN (?,?)`, Fingerprint(folderID), Fingerprint(alias)).Scan(&excluded)
				if err != nil {
					tx.Rollback()
					return count, err
				}
				if excluded > 0 {
					tx.Rollback()
					continue
				}
				id := folderID
				var known string
				err = tx.QueryRowContext(ctx, `SELECT character_id FROM character_files WHERE folder_id=?`, folderID).Scan(&known)
				if err == nil {
					id = known
				} else if !errors.Is(err, sql.ErrNoRows) {
					tx.Rollback()
					return count, err
				}
				if _, err = tx.ExecContext(ctx, `INSERT INTO characters(id,name,realm) VALUES(?,?,?) ON CONFLICT(id) DO NOTHING`, id, char.Name(), realm.Name()); err == nil {
					_, err = tx.ExecContext(ctx, `INSERT INTO character_files(folder_id,character_id) VALUES(?,?) ON CONFLICT(folder_id) DO NOTHING`, folderID, id)
				}
				if err != nil {
					tx.Rollback()
					return count, err
				}
				if alias != "" {
					if err = bindCharacterFolder(ctx, tx, folderID, id, alias, char.Name()); err != nil {
						tx.Rollback()
						return count, err
					}
				}
				if err = tx.Commit(); err != nil {
					return count, err
				}
				count++
			}
		}
	}
	return count, nil
}
func (s *Store) journalCharacter(ctx context.Context, alias, name, realm string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `SELECT character_id FROM journal_aliases WHERE alias=?`, alias).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	return alias, nil
}

func readCharacterIdentity(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 16<<10 {
		return "", errors.New("invalid Jornia identity file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if err != nil {
		return "", err
	}
	return parseCharacterIdentity(string(b))
}

func bindCharacterFolder(ctx context.Context, tx *sql.Tx, folder, current, alias, name string) error {
	var canonical string
	err := tx.QueryRowContext(ctx, `SELECT character_id FROM journal_aliases WHERE alias=?`, alias).Scan(&canonical)
	if errors.Is(err, sql.ErrNoRows) {
		var links int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM journal_aliases WHERE character_id=?`, current).Scan(&links); err != nil {
			return err
		}
		canonical = alias
		// Keep the preliminary ID, and therefore its drafts and selection, only
		// while it hasn't already been claimed by a different GUID.
		if current == folder && links == 0 {
			canonical = current
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO characters(id,name,realm) SELECT ?,name,realm FROM characters WHERE id=? ON CONFLICT DO NOTHING`, canonical, current); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO journal_aliases(alias,character_id) VALUES(?,?)`, alias, canonical); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if canonical != current && current == folder {
		// Repair the 0.5.1 duplicate only if the preliminary row has no private
		// content. Characters with independent journals, profiles or conversations
		// are never guessed/merged or deleted.
		var content int
		if err = tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM events WHERE character_id=?)+(SELECT count(*) FROM messages WHERE character_id=?)+(SELECT count(*) FROM character_profiles WHERE character_id=?)+(SELECT count(*) FROM journal_aliases WHERE character_id=?)+(SELECT count(*) FROM operations WHERE character_id=?)`, current, current, current, current, current).Scan(&content); err != nil {
			return err
		}
		if content == 0 {
			if _, err = tx.ExecContext(ctx, `UPDATE character_files SET character_id=? WHERE character_id=?`, canonical, current); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `DELETE FROM characters WHERE id=?`, current); err != nil {
				return err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE character_files SET character_id=? WHERE folder_id=?`, canonical, folder); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE characters SET name=? WHERE id=?`, name, canonical)
	return err
}
