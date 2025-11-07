package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eugenenazirov/re-partners/internal/storage"
)

const (
	defaultPort = "8080"
)

// Config aggregates runtime configuration resolved from environment variables.
type Config struct {
	Port                 string
	InitialPackSizes     []int
	ShutdownGracePeriod  time.Duration
	ReadHeaderTimeout    time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	EnableRequestLogging bool
}

// Load extracts configuration from the current process environment.
func Load() (Config, error) {
	cfg := Config{
		Port:                 defaultPort,
		InitialPackSizes:     storage.DefaultPackSizes(),
		ShutdownGracePeriod:  10 * time.Second,
		ReadHeaderTimeout:    5 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
		EnableRequestLogging: true,
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		cfg.Port = port
	}

	if rawSizes := strings.TrimSpace(os.Getenv("PACK_SIZES")); rawSizes != "" {
		sizes, err := parsePackSizes(rawSizes)
		if err != nil {
			return Config{}, fmt.Errorf("parse PACK_SIZES: %w", err)
		}
		cfg.InitialPackSizes = sizes
	}

	return cfg, nil
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
		sizes = append(sizes, value)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("no pack sizes provided")
	}
	return sizes, nil
}
