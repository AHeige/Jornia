package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AHeige/Jornia/internal/wisp"
)

func Handler(service *wisp.Service, ping func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if ping(ctx) != nil {
			write(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/wisp/ask", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req wisp.Request
		if err := decoder.Decode(&req); err != nil {
			write(w, 400, map[string]string{"error": "invalid JSON request"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			write(w, 400, map[string]string{"error": "expected one JSON object"})
			return
		}
		if err := wisp.ValidateJournal(req.Journal); err != nil {
			write(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := wisp.ValidateCharacter(req.Character); err != nil {
			write(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := wisp.ValidateSnapshot(req.Snapshot); err != nil {
			write(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if (len(req.Journal) > 0 || req.Character != nil || req.Snapshot != nil) && req.Mode != "generate" {
			write(w, 400, map[string]string{"error": "journal requires AI mode"})
			return
		}
		req.Question = strings.TrimSpace(req.Question)
		if len(req.Question) == 0 || len(req.Question) > 1000 || len(req.GameVersion) > 100 {
			write(w, 400, map[string]string{"error": "question must be 1–1000 bytes; game_version at most 100 bytes"})
			return
		}
		if req.GameVersion == "" {
			req.GameVersion = "forever"
		}
		if (req.Character != nil || req.Snapshot != nil) && req.GameVersion != "forever" {
			write(w, 400, map[string]string{"error": "character context belongs to Forever"})
			return
		}
		if req.Mode == "" {
			req.Mode = "auto"
		}
		if req.Mode != "auto" && req.Mode != "generate" {
			write(w, 400, map[string]string{"error": "mode must be auto or generate"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		answer, err := service.Ask(ctx, req)
		if err != nil {
			slog.Warn("ask failed", "mode", req.Mode, "game_version", req.GameVersion, "error", err.Error())
			switch {
			case errors.Is(err, wisp.ErrUnavailable):
				write(w, 503, map[string]string{"error": err.Error()})
			case errors.Is(err, wisp.ErrRateLimited):
				w.Header().Set("Retry-After", "60")
				write(w, 429, map[string]string{"error": err.Error()})
			default:
				write(w, 502, map[string]string{"error": "knowledge or inference request failed"})
			}
			return
		}
		var model, cost, inputTokens, outputTokens any
		if answer.Model != nil {
			model = *answer.Model
		}
		if answer.EstimatedCostEUR != nil {
			cost = *answer.EstimatedCostEUR
		}
		if answer.Usage != nil {
			inputTokens = answer.Usage.PromptTokens
			outputTokens = answer.Usage.CompletionTokens
		}
		slog.Info("ask completed", "status", answer.Status, "route", answer.Route, "game_version", answer.GameVersion, "latency_ms", answer.LatencyMS, "model", model, "prompt_tokens", inputTokens, "completion_tokens", outputTokens, "estimated_cost_eur", cost)
		write(w, 200, answer)
	})
	mux.HandleFunc("GET /api/knowledge/coverage", func(w http.ResponseWriter, r *http.Request) {
		reporter, ok := service.Store.(interface {
			Coverage(context.Context, string) (map[string]any, error)
		})
		if !ok {
			write(w, 503, map[string]string{"error": "coverage unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		report, err := reporter.Coverage(ctx, "forever")
		if err != nil {
			write(w, 503, map[string]string{"error": "coverage unavailable"})
			return
		}
		write(w, 200, report)
	})
	mux.HandleFunc("GET /api/knowledge", func(w http.ResponseWriter, r *http.Request) {
		version := r.URL.Query().Get("game_version")
		if version == "" {
			version = "forever"
		}
		if len(version) > 100 {
			write(w, 400, map[string]string{"error": "invalid game_version"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		g, err := service.Store.Graph(ctx, version, false)
		if err != nil {
			write(w, 503, map[string]string{"error": "knowledge unavailable"})
			return
		}
		write(w, 200, g)
	})
	return mux
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
