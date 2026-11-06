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

	"github.com/jackc/pgx/v5/pgxpool"
	"wisp/internal/ingestion"
	"wisp/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	max := flag.Int("max-requests", 200, "maximum public game-data requests in one resumable batch")
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
	worker := ingestion.Worker{Store: store, Client: client, MaxRequests: *max}
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
