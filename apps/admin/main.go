package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	"time"
	"wisp/internal/admin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Admin could not start:", err)
		os.Exit(1)
	}
}
func run() error {
	if os.Getenv("DATABASE_URL") == "" {
		return fmt.Errorf("admin database configuration missing")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("database configuration invalid")
	}
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	origin := os.Getenv("WISP_ADMIN_ORIGIN")
	if origin == "" {
		origin = "http://127.0.0.1:18983"
	}
	handler, err := admin.New(pool, origin, os.Getenv("WISP_ADMIN_PASSWORD"))
	if err != nil {
		return err
	}
	listen := os.Getenv("LISTEN_ADDR")
	if listen == "" {
		listen = "127.0.0.1:18091"
	}
	server := http.Server{Addr: listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	fmt.Println("Wisp invite admin ready on its private listener.")
	return server.ListenAndServe()
}
