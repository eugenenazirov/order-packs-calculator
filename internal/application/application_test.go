package application

import (
	"net/http"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/eugenenazirov/re-partners/internal/config"
	"go.uber.org/zap/zaptest"
)

func TestNewInitializesDependencies(t *testing.T) {
	cfg := baseTestConfig(":8085")
	cfg.InitialPackSizes = []int{400, 150}
	logger := zaptest.NewLogger(t)

	app, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	sizes, err := app.storage.GetPackSizes()
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}
	if want := []int{150, 400}; !slices.Equal(sizes, want) {
		t.Fatalf("expected pack sizes %v, got %v", want, sizes)
	}
	if app.server == nil || app.router == nil || app.handler == nil {
		t.Fatalf("expected server, router, and handler to be initialized")
	}
	if app.Server() != app.server {
		t.Fatalf("Server accessor did not return underlying instance")
	}
}

func TestNewServerAppliesConfig(t *testing.T) {
	cfg := baseTestConfig("9090")
	handler := http.NewServeMux()

	server := NewServer(cfg, handler)
	if server.Addr != ":9090" {
		t.Fatalf("expected address :9090, got %s", server.Addr)
	}
	if server.Handler != handler {
		t.Fatalf("expected handler to be applied")
	}
	if server.ReadHeaderTimeout != cfg.ReadHeaderTimeout ||
		server.WriteTimeout != cfg.WriteTimeout ||
		server.IdleTimeout != cfg.IdleTimeout {
		t.Fatalf("server timeouts do not match configuration")
	}
}

func TestResolveProjectPathFindsGoMod(t *testing.T) {
	path, err := resolveProjectPath("go.mod")
	if err != nil {
		t.Fatalf("resolveProjectPath returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected go.mod to exist at %s: %v", path, err)
	}
}

func TestNewReturnsErrorForInvalidPackSizes(t *testing.T) {
	cfg := baseTestConfig(":0")
	cfg.InitialPackSizes = nil

	if _, err := New(cfg, zaptest.NewLogger(t)); err == nil {
		t.Fatalf("expected error for invalid pack sizes")
	}
}

func TestResolveProjectPathUnknownTarget(t *testing.T) {
	if _, err := resolveProjectPath("definitely-not-a-real-file"); err == nil {
		t.Fatalf("expected error for missing resource")
	}
}

func baseTestConfig(port string) config.Config {
	return config.Config{
		Port:                 port,
		InitialPackSizes:     []int{250, 500},
		ShutdownGracePeriod:  50 * time.Millisecond,
		ReadHeaderTimeout:    20 * time.Millisecond,
		WriteTimeout:         30 * time.Millisecond,
		IdleTimeout:          40 * time.Millisecond,
		EnableRequestLogging: false,
		RateLimitRPS:         0,
		RateLimitBurst:       0,
	}
}
