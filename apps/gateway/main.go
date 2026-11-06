package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
	"wisp/internal/armory"
	"wisp/internal/httpapi"
)

func main() {
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("database configuration invalid")
	}
	config.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("database configuration invalid")
	}
	defer pool.Close()
	target, _ := url.Parse("http://api:8090")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) { http.Error(w, "Wisp is unavailable", 502) }
	server := http.Server{Addr: ":8092", Handler: httpapi.Public(proxy, pool, "/releases", &armory.Client{ClientID: os.Getenv("BLIZZARD_CLIENT_ID"), ClientSecret: os.Getenv("BLIZZARD_CLIENT_SECRET")}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192}
	log.Fatal(server.ListenAndServe())
}
