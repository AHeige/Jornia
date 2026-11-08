package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AHeige/Jornia/internal/ingestion"
	"github.com/AHeige/Jornia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	max := flag.Int("max-requests", 200, "maximum public game-data requests in one resumable batch")
	purge := flag.Bool("purge-only", false, "remove expired API evidence without contacting any provider")
	reproject := flag.Int("reproject-limit", 200, "maximum stale cached projections to rebuild without extending source expiry (at most 2000)")
	flag.Parse()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("invalid database configuration")
	}
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("database connection unavailable")
	}
	defer pool.Close()
	store := &storage.Store{Pool: pool}
	if err = store.Migrate(ctx); err != nil {
		return err
	}
	client := &ingestion.Client{Region: env("BLIZZARD_REGION", "eu"), Locale: env("BLIZZARD_LOCALE", "en_GB"), ClientID: os.Getenv("BLIZZARD_CLIENT_ID"), ClientSecret: os.Getenv("BLIZZARD_CLIENT_SECRET"), Interval: time.Second}
	worker := ingestion.Worker{Store: store, Client: client, MaxRequests: *max, ReprojectLimit: *reproject}
	if *purge {
		if err = worker.Purge(ctx); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "purged"})
	}
	report, err := worker.Run(ctx)
	_ = json.NewEncoder(os.Stdout).Encode(report)
	return err
}
func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
