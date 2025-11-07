package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("PACK_SIZES", "")

	cfg, err := Load()
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

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("PACK_SIZES", "10, 20 , 30")

	cfg, err := Load()
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
