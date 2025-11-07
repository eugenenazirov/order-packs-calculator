package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eugenenazirov/re-partners/internal/storage"
	"gopkg.in/yaml.v3"
)

const (
	defaultPort           = "8080"
	defaultRateLimitRPS   = 25.0
	defaultRateLimitBurst = 50
)

// Config aggregates runtime configuration resolved from multiple sources.
// Precedence: CLI flags > YAML config > Environment variables > Defaults
type Config struct {
	Port                 string        `yaml:"port"`
	InitialPackSizes     []int         `yaml:"pack_sizes"`
	ShutdownGracePeriod  time.Duration `yaml:"shutdown_grace_period"`
	ReadHeaderTimeout    time.Duration `yaml:"read_header_timeout"`
	WriteTimeout         time.Duration `yaml:"write_timeout"`
	IdleTimeout          time.Duration `yaml:"idle_timeout"`
	EnableRequestLogging bool          `yaml:"enable_request_logging"`
	RateLimitRPS         float64       `yaml:"-"`
	RateLimitBurst       int           `yaml:"-"`
}

// yamlConfig represents the YAML configuration file structure.
type yamlConfig struct {
	Port                 string        `yaml:"port"`
	PackSizes            []int         `yaml:"pack_sizes"`
	ShutdownGracePeriod  string        `yaml:"shutdown_grace_period"`
	ReadHeaderTimeout    string        `yaml:"read_header_timeout"`
	WriteTimeout         string        `yaml:"write_timeout"`
	IdleTimeout          string        `yaml:"idle_timeout"`
	EnableRequestLogging bool          `yaml:"enable_request_logging"`
	RateLimit            yamlRateLimit `yaml:"rate_limit"`
}

// yamlRateLimit represents the rate limit section in YAML.
type yamlRateLimit struct {
	RPS   float64 `yaml:"rps"`
	Burst int     `yaml:"burst"`
}

// CLIOverrides holds command-line flag overrides.
type CLIOverrides struct {
	ConfigFile     string
	Port           *string
	PackSizesStr   *string
	RateLimitRPS   *float64
	RateLimitBurst *int
}

// Load extracts configuration from multiple sources with precedence:
// CLI flags > YAML config > Environment variables > Defaults
func Load(overrides *CLIOverrides) (Config, error) {
	cfg := defaultConfig()

	// Load from YAML file if specified
	if overrides != nil && overrides.ConfigFile != "" {
		yamlCfg, err := loadFromFile(overrides.ConfigFile)
		if err != nil {
			return Config{}, fmt.Errorf("load YAML config: %w", err)
		}
		applyYAMLConfig(&cfg, yamlCfg)
	}

	// Apply environment variables (override YAML)
	applyEnvConfig(&cfg)

	// Apply CLI overrides (highest precedence)
	if overrides != nil {
		if err := applyCLIOverrides(&cfg, overrides); err != nil {
			return Config{}, err
		}
	}

	// Validate final configuration
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// defaultConfig returns a Config with default values.
func defaultConfig() Config {
	return Config{
		Port:                 defaultPort,
		InitialPackSizes:     storage.DefaultPackSizes(),
		ShutdownGracePeriod:  10 * time.Second,
		ReadHeaderTimeout:    5 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
		EnableRequestLogging: true,
		RateLimitRPS:         defaultRateLimitRPS,
		RateLimitBurst:       defaultRateLimitBurst,
	}
}

// loadFromFile loads configuration from a YAML file.
func loadFromFile(path string) (*yamlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var yamlCfg yamlConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &yamlCfg, nil
}

// applyYAMLConfig applies YAML configuration to the Config struct.
func applyYAMLConfig(cfg *Config, yamlCfg *yamlConfig) {
	if yamlCfg.Port != "" {
		cfg.Port = yamlCfg.Port
	}

	if len(yamlCfg.PackSizes) > 0 {
		cfg.InitialPackSizes = yamlCfg.PackSizes
	}

	if yamlCfg.ShutdownGracePeriod != "" {
		if d, err := time.ParseDuration(yamlCfg.ShutdownGracePeriod); err == nil {
			cfg.ShutdownGracePeriod = d
		}
	}

	if yamlCfg.ReadHeaderTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.ReadHeaderTimeout); err == nil {
			cfg.ReadHeaderTimeout = d
		}
	}

	if yamlCfg.WriteTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.WriteTimeout); err == nil {
			cfg.WriteTimeout = d
		}
	}

	if yamlCfg.IdleTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.IdleTimeout); err == nil {
			cfg.IdleTimeout = d
		}
	}

	cfg.EnableRequestLogging = yamlCfg.EnableRequestLogging

	if yamlCfg.RateLimit.RPS >= 0 {
		cfg.RateLimitRPS = yamlCfg.RateLimit.RPS
	}

	if yamlCfg.RateLimit.Burst >= 0 {
		cfg.RateLimitBurst = yamlCfg.RateLimit.Burst
	}
}

// applyEnvConfig applies environment variable configuration.
func applyEnvConfig(cfg *Config) {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		cfg.Port = port
	}

	if rawSizes := strings.TrimSpace(os.Getenv("PACK_SIZES")); rawSizes != "" {
		sizes, err := parsePackSizes(rawSizes)
		if err == nil && len(sizes) > 0 {
			cfg.InitialPackSizes = sizes
		}
	}

	if rps := strings.TrimSpace(os.Getenv("RATE_LIMIT_RPS")); rps != "" {
		if value, err := strconv.ParseFloat(rps, 64); err == nil && value >= 0 {
			cfg.RateLimitRPS = value
		}
	}

	if burst := strings.TrimSpace(os.Getenv("RATE_LIMIT_BURST")); burst != "" {
		if value, err := strconv.Atoi(burst); err == nil && value >= 0 {
			cfg.RateLimitBurst = value
		}
	}
}

// applyCLIOverrides applies command-line flag overrides.
func applyCLIOverrides(cfg *Config, overrides *CLIOverrides) error {
	if overrides.Port != nil && *overrides.Port != "" {
		cfg.Port = *overrides.Port
	}

	if overrides.PackSizesStr != nil && *overrides.PackSizesStr != "" {
		sizes, err := parsePackSizes(*overrides.PackSizesStr)
		if err != nil {
			return fmt.Errorf("parse pack sizes: %w", err)
		}
		cfg.InitialPackSizes = sizes
	}

	if overrides.RateLimitRPS != nil && *overrides.RateLimitRPS >= 0 {
		cfg.RateLimitRPS = *overrides.RateLimitRPS
	}

	if overrides.RateLimitBurst != nil && *overrides.RateLimitBurst >= 0 {
		cfg.RateLimitBurst = *overrides.RateLimitBurst
	}

	return nil
}

// validateConfig validates the final configuration.
func validateConfig(cfg Config) error {
	if cfg.RateLimitRPS < 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be >= 0")
	}
	if cfg.RateLimitBurst < 0 {
		return fmt.Errorf("RATE_LIMIT_BURST must be >= 0")
	}
	if len(cfg.InitialPackSizes) == 0 {
		return fmt.Errorf("pack sizes cannot be empty")
	}
	return nil
}

// parsePackSizes parses a comma-separated string of pack sizes into a slice of integers.
// It validates that all values are positive integers.
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
