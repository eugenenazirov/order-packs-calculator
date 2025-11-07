package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"go.uber.org/zap"

	"github.com/eugenenazirov/re-partners/internal/application"
	"github.com/eugenenazirov/re-partners/internal/config"
	"github.com/eugenenazirov/re-partners/internal/logging"
)

var signalNotify = signal.Notify

func main() {
	kingpinApp := kingpin.New("pack-calculator", "Order Packs Calculator - determines minimal packs needed to fulfil orders")
	configFile := kingpinApp.Flag("config", "Path to YAML configuration file").String()
	port := kingpinApp.Flag("port", "HTTP port exposed by the service").String()
	packSizesStr := kingpinApp.Flag("pack-sizes", "Comma-separated initial pack sizes").String()
	rateLimitRPSFlag := kingpinApp.Flag("rate-limit-rps", "Requests per second allowed (set 0 to disable)").Default("-1").Float64()
	rateLimitBurstFlag := kingpinApp.Flag("rate-limit-burst", "Burst capacity for rate limiter (set 0 to disable)").Default("-1").Int()

	kingpin.MustParse(kingpinApp.Parse(os.Args[1:]))

	overrides := &config.CLIOverrides{
		ConfigFile: *configFile,
	}

	if *port != "" {
		overrides.Port = port
	}

	if *packSizesStr != "" {
		overrides.PackSizesStr = packSizesStr
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

	app, err := application.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}

	if err := app.Start(); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}

	shutdown(app.Server(), cfg.ShutdownGracePeriod, logger)
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
