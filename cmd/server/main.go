package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/eugenenazirov/re-partners/internal/api"
	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/config"
	"github.com/eugenenazirov/re-partners/internal/logging"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

var signalNotify = signal.Notify

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}

	logger, err := logging.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer func() {
		_ = logger.Sync()
	}()

	store := storage.NewMemoryStorage()
	if err := store.SetPackSizes(cfg.InitialPackSizes); err != nil {
		logger.Fatal("failed to apply initial pack sizes", zap.Error(err))
	}

	calc := calculator.New()
	handler := api.NewHandler(calc, store)
	apiRouter := api.NewRouter(handler, logger, api.WithLogging(cfg.EnableRequestLogging))

	rootHandler, err := buildRootHandler(apiRouter)
	if err != nil {
		logger.Fatal("failed to build HTTP handler", zap.Error(err))
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
		logger.Info("server listening", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	shutdown(server, cfg.ShutdownGracePeriod, logger)
}

func shutdown(server *http.Server, timeout time.Duration, logger *zap.Logger) {
	quit := make(chan os.Signal, 1)
	signalNotify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Warn("graceful shutdown failed", zap.Error(err))
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("forced close failed", zap.Error(closeErr))
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
