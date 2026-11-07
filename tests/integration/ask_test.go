package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"wisp/internal/ai"
	"wisp/internal/httpapi"
	"wisp/internal/storage"
	"wisp/internal/wisp"
)

func TestAshenvaleStack(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("wisp_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &storage.Store{Pool: pool}
	for i := 0; i < 2; i++ {
		if err = store.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		if err = store.Seed(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM entities").Scan(&count); err != nil || count != 55 {
		t.Fatalf("idempotent seed count=%d error=%v", count, err)
	}
	var calls int
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if strings.Contains(request.Messages[1].Content, "Hidden Story") {
			t.Error("spoiler sent to inference")
		}
		result := ai.Result{Status: "answered", Text: "The fixture links Astranaar to Ashenvale.", EvidenceIDs: []string{"edge:astranaar-in-ashenvale"}}
		if strings.Contains(request.Messages[1].Content, "fabricate") {
			result.EvidenceIDs = []string{"entity:not-real"}
		}
		content, _ := json.Marshal(result)
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "test-model", "usage": ai.Usage{PromptTokens: 100, CompletionTokens: 20}, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": string(content)}}}})
	}))
	defer provider.Close()
	price := 0.3
	service := &wisp.Service{Store: store, Generator: &ai.Berget{BaseURL: provider.URL, Model: "test-model", APIKey: "test", Client: provider.Client()}, InputPrice: &price, OutputPrice: &price}
	handler := httpapi.Handler(service, pool.Ping)
	ask := func(body string) (int, wisp.Answer) {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/wisp/ask", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var answer wisp.Answer
		_ = json.Unmarshal(w.Body.Bytes(), &answer)
		return w.Code, answer
	}
	for _, tc := range []struct {
		name, body, status, contains string
		code                         int
	}{
		{"location", `{"question":"Where is Astranaar?","game_version":"fixture-ashenvale-v1"}`, "answered", "Ashenvale", 200},
		{"zone", `{"question":"Where is Ashenvale?","game_version":"fixture-ashenvale-v1"}`, "answered", "Kalimdor", 200},
		{"places", `{"question":"What places are in Ashenvale?","game_version":"fixture-ashenvale-v1"}`, "answered", "The Zoram Strand", 200},
		{"forever default", `{"question":"Where is Astranaar?"}`, "answered", "Ashenvale", 200},
		{"unavailable version", `{"question":"Where is Astranaar?","game_version":"forever-unchecked-build"}`, "unknown", "verified", 200},
		{"expanded geography", `{"question":"Var ligger Mystral Lake?"}`, "answered", "Ashenvale", 200},
		{"real NPC", `{"question":"Where is Raene Wolfrunner?"}`, "answered", "Astranaar", 200},
		{"real giver", `{"question":"Who gives Raene's Cleansing?"}`, "answered", "Raene Wolfrunner", 200},
		{"real quests", `{"question":"What quests are in Ashenvale?"}`, "answered", "Raene's Cleansing", 200},
		{"nested location", `{"question":"Where is Blackfathom Deeps?"}`, "answered", "The Zoram Strand", 200},
		{"sv location", `{"question":"Var ligger Astranaar?"}`, "answered", "Ashenvale", 200},
		{"sv giver", `{"question":"Vem ger Raene's Cleansing?"}`, "answered", "Raene Wolfrunner", 200},
		{"no synthetic memory in forever", `{"question":"Tell me about Hidden Story","spoilers":true}`, "unknown", "verified", 200},
		{"no unverified levels", `{"question":"What level should I be in Ashenvale?"}`, "unknown", "verified", 200},
		{"unseeded unknown", `{"question":"Where is Orgrimmar?","game_version":"fixture-ashenvale-v1"}`, "unknown", "verified", 200},
		{"unsupported intent", `{"question":"What level should I be in Ashenvale?","game_version":"fixture-ashenvale-v1"}`, "unknown", "verified", 200},
		{"no memory", `{"question":"What did I do last night?","game_version":"fixture-ashenvale-v1"}`, "unknown", "verified", 200},
		{"spoiler blocked", `{"question":"Tell me about Hidden Story","game_version":"fixture-ashenvale-v1"}`, "unknown", "verified", 200},
		{"spoiler consent", `{"question":"Tell me about Hidden Story","game_version":"fixture-ashenvale-v1","spoilers":true}`, "answered", "synthetic", 200},
		{"empty", `{"question":" "}`, "", "", 400},
		{"extra object", `{"question":"Hi"} {}`, "", "", 400},
		{"unknown field", `{"question":"Hi","private_key":"x"}`, "", "", 400},
		{"invalid mode", `{"question":"Hi","mode":"bad"}`, "", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, a := ask(tc.body)
			if code != tc.code || a.Status != tc.status || !strings.Contains(a.Text, tc.contains) {
				t.Fatalf("code %d answer %+v", code, a)
			}
			if a.Status == "answered" && (len(a.Sources) == 0 || len(a.EvidenceIDs) == 0) {
				t.Fatal("missing provenance")
			}
			if a.Model != nil {
				t.Fatal("unexpected inference")
			}
			if a.Status == "answered" && len(a.Warnings) == 0 {
				t.Fatal("source assurance warning missing")
			}
		})
	}

	// Sources must retain review assurance and attribution in the actual API response.
	_, reviewed := ask(`{"question":"Where is Raene Wolfrunner?"}`)
	if len(reviewed.Sources) == 0 || strings.Contains(reviewed.Text, "fixture") {
		t.Fatalf("reviewed data mislabelled: %+v", reviewed)
	}
	for _, src := range reviewed.Sources {
		if src.ValidationStatus != "source_checked" || src.Provenance == nil || src.Provenance.CheckedAt != "2026-11-05" || src.Provenance.Attribution == "" || len(src.Provenance.ReviewSHA256) != 64 {
			t.Fatalf("review metadata missing: %+v", src)
		}
	}
	_, places := ask(`{"question":"What places are in Ashenvale?"}`)
	if !strings.Contains(places.Text, "Blackfathom Deeps") || strings.Contains(places.Text, "Raene Wolfrunner") {
		t.Fatalf("location traversal incorrect: %s", places.Text)
	}
	// Unreviewed rows must not enter retrieval, deterministic answers, or model context.
	if _, err = pool.Exec(ctx, `INSERT INTO sources(id,title,uri,revision,license,validation_status) VALUES('unreviewed','Unreviewed','wisp:test','1','test','unverified'); INSERT INTO entities(id,game_version,kind,name,description,source_id) VALUES('unreviewed-place','forever','location','Unreviewed Place','Must not be served','unreviewed')`); err != nil {
		t.Fatal(err)
	}
	_, blocked := ask(`{"question":"Tell me about Unreviewed Place"}`)
	if blocked.Status != "unknown" {
		t.Fatal("unreviewed data exposed")
	}
	if calls != 0 {
		t.Fatal("deterministic requests invoked inference")
	}
	code, a := ask(`{"question":"Briefly explain Astranaar's location","game_version":"fixture-ashenvale-v1","mode":"generate"}`)
	if code != 200 || a.Status != "answered" || a.Model == nil || *a.Model != "test-model" || a.Usage == nil || a.EstimatedCostEUR == nil || *a.EstimatedCostEUR < 0.0000359 || *a.EstimatedCostEUR > 0.0000361 {
		t.Fatalf("bad generation result %d %+v", code, a)
	}
	code, a = ask(`{"question":"Astranaar fabricate evidence","game_version":"fixture-ashenvale-v1","mode":"generate"}`)
	if code != 200 || a.Status != "unknown" || len(a.Sources) != 0 || len(a.EvidenceIDs) != 0 || a.Usage == nil {
		t.Fatalf("unvalidated answer leaked: %+v", a)
	}
	if calls != 2 {
		t.Fatalf("expected 2 inference calls, got %d", calls)
	}
	code, a = ask(`{"question":"Tell me about unknownplace","game_version":"fixture-ashenvale-v1","mode":"generate"}`)
	if code != 200 || calls != 3 || a.Status != "unknown" || len(a.EvidenceIDs) != 0 {
		t.Fatal("explicit generation without knowledge must remain unknown")
	}
	service.Generator = nil
	code, _ = ask(`{"question":"Astranaar","game_version":"fixture-ashenvale-v1","mode":"generate"}`)
	if code != 503 {
		t.Fatal("unconfigured provider must return 503")
	}
	r := httptest.NewRequest("GET", "/api/knowledge?game_version=fixture-ashenvale-v1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte("Hidden Story")) {
		t.Fatal("knowledge endpoint leaked spoiler")
	}
	r = httptest.NewRequest("GET", "/api/knowledge/coverage", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var coverage struct {
		Complete   bool                        `json:"complete"`
		Entities   int                         `json:"entities"`
		Categories []storage.KnowledgeCategory `json:"categories"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &coverage); err != nil || w.Code != 200 || coverage.Complete || coverage.Entities != 49 || len(coverage.Categories) != 25 {
		t.Fatalf("misleading coverage %s", w.Body.String())
	}
	r = httptest.NewRequest("GET", "/healthz", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("health failed")
	}
}
