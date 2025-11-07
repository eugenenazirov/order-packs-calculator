package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"go.uber.org/zap"

	"github.com/eugenenazirov/re-partners/internal/api"
	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/config"
	"github.com/eugenenazirov/re-partners/internal/logging"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

var signalNotify = signal.Notify

func main() {
	app := kingpin.New("pack-calculator", "Order Packs Calculator - determines minimal packs needed to fulfil orders")
	configFile := app.Flag("config", "Path to YAML configuration file").String()
	port := app.Flag("port", "HTTP port exposed by the service").String()
	packSizesStr := app.Flag("pack-sizes", "Comma-separated initial pack sizes").String()
	rateLimitRPSFlag := app.Flag("rate-limit-rps", "Requests per second allowed (set 0 to disable)").Default("-1").Float64()
	rateLimitBurstFlag := app.Flag("rate-limit-burst", "Burst capacity for rate limiter (set 0 to disable)").Default("-1").Int()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	overrides := &config.CLIOverrides{
		ConfigFile: *configFile,
	}

	if *port != "" {
		overrides.Port = port
	}

	if *packSizesStr != "" {
		sizes, err := parsePackSizes(*packSizesStr)
		if err != nil {
			panic(fmt.Sprintf("failed to parse pack sizes: %v", err))
		}
		overrides.PackSizes = &sizes
	}

	if *rateLimitRPSFlag >= 0 {
		overrides.RateLimitRPS = rateLimitRPSFlag
	}

	if *rateLimitBurstFlag >= 0 {
		overrides.RateLimitBurst = rateLimitBurstFlag
	}

	cfg, err := config.Load(overrides)
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
	apiRouter := api.NewRouter(handler, logger,
		api.WithLogging(cfg.EnableRequestLogging),
		api.WithRateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst),
	)

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

func parsePackSizes(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	sizes := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", part)
		}
		if value <= 0 {
			return nil, fmt.Errorf("pack size must be positive, got %d", value)
		}
		sizes = append(sizes, value)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("no pack sizes provided")
	}
	return sizes, nil
}
