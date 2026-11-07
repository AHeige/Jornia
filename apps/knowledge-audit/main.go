// Read-only coverage and retrieval latency audit. No external/model requests.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"sort"
	"time"
	"wisp/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Knowledge audit failed:", err)
		os.Exit(1)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("invalid database configuration")
	}
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := &storage.Store{Pool: pool}
	coverage, err := store.Coverage(ctx, "forever")
	if err != nil {
		return err
	}
	tests := []map[string]any{}
	for _, q := range []string{"Who gives The Damned?", "Where is Rattling the Rattlecages?", "Wand Specialization", "elixir minor strength", "Mage", "wisp_nonexistent_quest_927184"} {
		times := []float64{}
		anchors, entities := 0, 0
		for i := 0; i < 21; i++ {
			start := time.Now()
			g, e := store.Retrieve(ctx, "forever", false, q)
			if e != nil {
				return e
			}
			if i > 0 {
				times = append(times, float64(time.Since(start).Microseconds())/1000)
			}
			anchors = len(g.Anchors)
			entities = len(g.Entities)
		}
		sort.Float64s(times)
		tests = append(tests, map[string]any{"question": q, "samples": len(times), "median_ms": times[len(times)/2], "p95_ms": times[len(times)*95/100-1], "anchors": anchors, "entities": entities})
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"coverage": coverage, "retrieval": tests, "measurement": "Go Store.Retrieve including source/build filters and two-hop graph; excludes network, OCR and inference"})
}
