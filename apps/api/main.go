package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"wisp/internal/ai"
	"wisp/internal/httpapi"
	"wisp/internal/storage"
	"wisp/internal/wisp"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "healthcheck" {
		client := http.Client{Timeout: 2 * time.Second}
		res, err := client.Get("http://127.0.0.1:8090/healthz")
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return errors.New("unhealthy")
		}
		return nil
	}
	if command != "serve" && command != "bootstrap" {
		return fmt.Errorf("unknown command %q", command)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return errors.New("invalid database configuration")
	}
	defer pool.Close()
	start, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = pool.Ping(start); err != nil {
		return errors.New("database connection failed")
	}
	store := &storage.Store{Pool: pool}
	if command == "bootstrap" {
		if err = store.Migrate(start); err != nil {
			return err
		}
		if err = store.Seed(start); err != nil {
			return err
		}
		log.Print("migrations, Ashenvale fixture, and reviewed Forever knowledge ready")
		return nil
	}
	service := &wisp.Service{Store: store}
	if gateway := os.Getenv("LANGCHAIN_URL"); gateway != "" {
		parsed, err := url.Parse(gateway)
		if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "model" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("LANGCHAIN_URL must identify the private model service")
		}
		service.Generator = &ai.Berget{BaseURL: gateway, APIKey: "internal", Model: env("BERGET_MODEL", "mistralai/Mistral-Small-3.2-24B-Instruct-2506")}
		service.InputPrice, err = price("BERGET_INPUT_EUR_PER_MILLION")
		if err != nil {
			return err
		}
		service.OutputPrice, err = price("BERGET_OUTPUT_EUR_PER_MILLION")
		if err != nil {
			return err
		}
		log.Print("Legacy model proxy enabled; Wisp tool orchestration is owned by Go")
	} else if key := os.Getenv("BERGET_API_KEY"); key != "" {
		base := env("BERGET_BASE_URL", "https://api.berget.ai/v1")
		u, err := url.Parse(base)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("BERGET_BASE_URL must be an HTTPS URL without credentials, query, or fragment")
		}
		service.Generator = &ai.Berget{BaseURL: base, APIKey: key, Model: env("BERGET_MODEL", "mistralai/Mistral-Small-3.2-24B-Instruct-2506")}
		service.InputPrice, err = price("BERGET_INPUT_EUR_PER_MILLION")
		if err != nil {
			return err
		}
		service.OutputPrice, err = price("BERGET_OUTPUT_EUR_PER_MILLION")
		if err != nil {
			return err
		}
		log.Print("Berget inference enabled; limit 6 calls/minute and 2 concurrent calls")
	} else {
		log.Print("Berget key absent; deterministic answers available")
	}
	server := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8090"), Handler: httpapi.Handler(service, pool.Ping), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192}
	done := make(chan error, 1)
	go func() { log.Printf("Wisp listening on %s", server.Addr); done <- server.ListenAndServe() }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func price(key string) (*float64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("%s must be a finite nonnegative number", key)
	}
	return &v, nil
}
