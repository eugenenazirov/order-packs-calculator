package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/eugenenazirov/re-partners/internal/api"
	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/config"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	store := storage.NewMemoryStorage()
	if err := store.SetPackSizes(cfg.InitialPackSizes); err != nil {
		log.Fatalf("failed to apply initial pack sizes: %v", err)
	}

	calc := calculator.New()
	handler := api.NewHandler(calc, store)
	apiRouter := api.NewRouter(handler, api.WithLogging(cfg.EnableRequestLogging))

	rootHandler, err := buildRootHandler(apiRouter)
	if err != nil {
		log.Fatalf("failed to build HTTP handler: %v", err)
	}

	addr := cfg.Port
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           rootHandler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	go func() {
		log.Printf("Order Packs Calculator listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	shutdown(server, cfg.ShutdownGracePeriod)
}

func shutdown(server *http.Server, timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("forced close failed: %v", closeErr)
		}
	}
}

func buildRootHandler(apiHandler http.Handler) (http.Handler, error) {
	mux := http.NewServeMux()

	staticPath, err := resolveProjectPath(filepath.Join("web", "static"))
	if err != nil {
		return nil, err
	}
	staticDir := http.Dir(staticPath)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticDir)))
	mux.Handle("/api/", apiHandler)

	indexPath, err := resolveProjectPath(filepath.Join("web", "templates", "index.html"))
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	}))

	return mux, nil
}

func resolveProjectPath(relative string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("unable to locate %s", relative)
}
