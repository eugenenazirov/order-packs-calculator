package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("PACK_SIZES", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %s, got %s", defaultPort, cfg.Port)
	}
	if len(cfg.InitialPackSizes) == 0 {
		t.Fatalf("expected default pack sizes, got none")
	}
	if cfg.ShutdownGracePeriod != 10*time.Second {
		t.Fatalf("unexpected shutdown grace period: %s", cfg.ShutdownGracePeriod)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("PACK_SIZES", "10, 20 , 30")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != "9000" {
		t.Fatalf("expected overridden port, got %s", cfg.Port)
	}
	if want := []int{10, 20, 30}; len(cfg.InitialPackSizes) != len(want) {
		t.Fatalf("unexpected pack sizes: %v", cfg.InitialPackSizes)
	}
}

func TestLoadYAMLConfig(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("PACK_SIZES", "")

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: "9090"
pack_sizes:
  - 100
  - 200
  - 300
shutdown_grace_period: "20s"
rate_limit:
  rps: 50.0
  burst: 100
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	overrides := &CLIOverrides{
		ConfigFile: yamlFile,
	}

	cfg, err := Load(overrides)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("expected port from YAML 9090, got %s", cfg.Port)
	}
	if want := []int{100, 200, 300}; len(cfg.InitialPackSizes) != len(want) {
		t.Fatalf("unexpected pack sizes: %v", cfg.InitialPackSizes)
	}
	if cfg.ShutdownGracePeriod != 20*time.Second {
		t.Fatalf("expected shutdown grace period 20s, got %s", cfg.ShutdownGracePeriod)
	}
	if cfg.RateLimitRPS != 50.0 {
		t.Fatalf("expected rate limit RPS 50.0, got %f", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 100 {
		t.Fatalf("expected rate limit burst 100, got %d", cfg.RateLimitBurst)
	}
}

func TestLoadPrecedence_CLIOverridesYAML(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("PACK_SIZES", "")

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: "9090"
pack_sizes:
  - 100
  - 200
rate_limit:
  rps: 50.0
  burst: 100
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	port := "8080"
	packSizesStr := "10,20,30"
	rps := 25.0
	burst := 50

	overrides := &CLIOverrides{
		ConfigFile:     yamlFile,
		Port:           &port,
		PackSizesStr:   &packSizesStr,
		RateLimitRPS:   &rps,
		RateLimitBurst: &burst,
	}

	cfg, err := Load(overrides)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// CLI should override YAML
	if cfg.Port != "8080" {
		t.Fatalf("expected CLI port 8080, got %s", cfg.Port)
	}
	if want := []int{10, 20, 30}; len(cfg.InitialPackSizes) != len(want) {
		t.Fatalf("expected CLI pack sizes %v, got %v", want, cfg.InitialPackSizes)
	}
	if cfg.RateLimitRPS != 25.0 {
		t.Fatalf("expected CLI rate limit RPS 25.0, got %f", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 50 {
		t.Fatalf("expected CLI rate limit burst 50, got %d", cfg.RateLimitBurst)
	}
}

func TestLoadPrecedence_EnvOverridesYAML(t *testing.T) {
	t.Setenv("PORT", "7000")
	t.Setenv("PACK_SIZES", "5,10,15")

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: "9090"
pack_sizes:
  - 100
  - 200
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	overrides := &CLIOverrides{
		ConfigFile: yamlFile,
	}

	cfg, err := Load(overrides)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Environment should override YAML
	if cfg.Port != "7000" {
		t.Fatalf("expected env port 7000, got %s", cfg.Port)
	}
	if want := []int{5, 10, 15}; len(cfg.InitialPackSizes) != len(want) {
		t.Fatalf("expected env pack sizes %v, got %v", want, cfg.InitialPackSizes)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `port: "9090"
invalid: [unclosed bracket
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	overrides := &CLIOverrides{
		ConfigFile: yamlFile,
	}

	_, err := Load(overrides)
	if err == nil {
		t.Fatalf("expected error for invalid YAML, got nil")
	}
}

func TestLoadNonExistentYAML(t *testing.T) {
	overrides := &CLIOverrides{
		ConfigFile: "/nonexistent/config.yaml",
	}

	_, err := Load(overrides)
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}

func TestParsePackSizes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := parsePackSizes("1,2,3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []int{1, 2, 3}; len(got) != len(want) {
			t.Fatalf("unexpected sizes: %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := parsePackSizes(" , "); err == nil {
			t.Fatalf("expected error for empty string")
		}
		if _, err := parsePackSizes("1,a"); err == nil {
			t.Fatalf("expected error for invalid integer")
		}
	})
}
