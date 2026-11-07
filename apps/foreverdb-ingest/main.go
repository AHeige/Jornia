package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wisp/internal/foreverdb"
	"wisp/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	max := flag.Int("max-requests", 200, "maximum provider requests; daily shared safety cap is 8000")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("database configuration invalid")
	}
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	s := &storage.Store{Pool: pool}
	if err = s.Migrate(ctx); err != nil {
		return err
	}
	w := foreverdb.Worker{Store: s, Client: &foreverdb.Client{Key: os.Getenv("FOREVERDB_API_KEY"), DeveloperID: os.Getenv("FOREVERDB_DEVELOPER_ID")}, MaxRequests: *max}
	r, err := w.Run(ctx)
	json.NewEncoder(os.Stdout).Encode(r)
	return err
}
