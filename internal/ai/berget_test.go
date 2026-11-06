package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBergetContract(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantErr    bool
		wantUsage  bool
	}{
		{"ok", `{"model":"baseline","choices":[{"finish_reason":"stop","message":{"content":"{\"status\":\"answered\",\"text\":\"Fixture answer\",\"evidence_ids\":[\"entity:ashenvale\"]}"}}],"usage":{"prompt_tokens":100,"completion_tokens":20}}`, 200, false, true},
		{"missing usage", `{"model":"baseline","choices":[{"finish_reason":"stop","message":{"content":"{\"status\":\"unknown\",\"text\":\"Unknown\",\"evidence_ids\":[]}"}}]}`, 200, false, false},
		{"partial usage", `{"model":"baseline","choices":[{"finish_reason":"stop","message":{"content":"{\"status\":\"unknown\",\"text\":\"Unknown\",\"evidence_ids\":[]}"}}],"usage":{"prompt_tokens":100}}`, 200, false, false},
		{"unauthorized", `secret diagnostic`, 401, true, false},
		{"truncated", `{"model":"baseline","choices":[{"finish_reason":"length","message":{"content":"{}"}}]}`, 200, true, false},
		{"bad JSON", `{`, 200, true, false},
		{"no choices", `{"model":"baseline","choices":[]}`, 200, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("bad endpoint or authentication")
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if request["model"] != "baseline" || request["max_tokens"] != float64(220) {
					t.Error("missing model or output bound")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			b := Berget{BaseURL: server.URL, APIKey: "test-key", Model: "baseline", Client: server.Client()}
			result, err := b.Generate(context.Background(), "Where is Ashenvale?", "fixture")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Fatal("provider diagnostics leaked")
			}
			if err == nil && (result.Usage != nil) != tc.wantUsage {
				t.Fatal("incorrect usage availability")
			}
		})
	}
}
