package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"wisp/internal/companion"
)

//go:embed web/index.html web/app.js web/style.css web/media/*
var web embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	listen := flag.String("listen", "127.0.0.1:8787", "local loopback address")
	backend := flag.String("backend", "https://wisp.logosab.com", "central Wisp API: HTTPS or SSH-forwarded loopback HTTP")
	data := flag.String("data-dir", "", "local private storage directory")
	noBrowser := flag.Bool("no-browser", false, "do not open the browser automatically")
	vm := flag.Bool("vm", false, "connect through the dedicated private VM SSH tunnel")
	prepareVM := flag.Bool("prepare-vm", false, "create only a dedicated local Wisp SSH key and print its public-key path")
	flag.Parse()
	if *data == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		if runtime.GOOS == "windows" && os.Getenv("LOCALAPPDATA") != "" {
			base = os.Getenv("LOCALAPPDATA")
		}
		*data = filepath.Join(base, "Wisp")
	}
	connectionDir := filepath.Join(*data, "connection")
	if *prepareVM {
		public, err := companion.PrepareVMConnection(connectionDir)
		if err != nil {
			return err
		}
		fmt.Println("Public key:", public)
		fmt.Println("Private key stays on this computer; register only the .pub file on the VM.")
		return nil
	}
	if *vm {
		address, stop, err := companion.StartVMTunnel(connectionDir, os.Stderr)
		if err != nil {
			return err
		}
		defer stop()
		*backend = address
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("companion must listen on a numeric loopback address")
	}
	store, err := companion.Open(filepath.Join(*data, "player.db"))
	if err != nil {
		return err
	}
	defer store.DB.Close()
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	defer listener.Close()
	assets, _ := fs.Sub(web, "web")
	app, err := companion.NewServer(store, *backend, listener.Addr().String(), assets)
	if err != nil {
		return err
	}
	journal, err := companion.NewJournalSync(store)
	if err != nil {
		return err
	}
	app.Journal = journal
	journalContext, stopJournal := context.WithCancel(context.Background())
	defer stopJournal()
	go journal.Run(journalContext)
	server := &http.Server{Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192}
	address := "http://" + listener.Addr().String()
	fmt.Println("Wisp:", address)
	fmt.Println("Your local database:", filepath.Join(*data, "player.db"))
	fmt.Println("Close this console to quit. No game history is sent automatically.")
	if !*noBrowser {
		go func() {
			switch runtime.GOOS {
			case "windows":
				_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", address).Start()
			case "darwin":
				_ = exec.Command("open", address).Start()
			default:
				_ = exec.Command("xdg-open", address).Start()
			}
		}()
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
