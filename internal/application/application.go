package application

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/eugenenazirov/re-partners/internal/api"
	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/config"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

// App encapsulates the application dependencies and HTTP server.
type App struct {
	storage    storage.Storage
	calculator calculator.Calculator
	handler    *api.Handler
	router     http.Handler
	logger     *zap.Logger
	server     *http.Server
}

// New initializes the application with all dependencies from the provided configuration.
func New(cfg config.Config, logger *zap.Logger) (*App, error) {
	store := storage.NewMemoryStorage()
	if err := store.SetPackSizes(cfg.InitialPackSizes); err != nil {
		return nil, fmt.Errorf("failed to apply initial pack sizes: %w", err)
	}

	calc := calculator.New()
	handler := api.NewHandler(calc, store)
	apiRouter := api.NewRouter(handler, logger,
		api.WithLogging(cfg.EnableRequestLogging),
		api.WithRateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst),
	)

	rootHandler, err := BuildRootHandler(apiRouter)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP handler: %w", err)
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

	return &App{
		storage:    store,
		calculator: calc,
		handler:    handler,
		router:     apiRouter,
		logger:     logger,
		server:     server,
	}, nil
}

// BuildRootHandler constructs the root HTTP handler that serves static files and routes API requests.
func BuildRootHandler(apiHandler http.Handler) (http.Handler, error) {
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

// NewServer creates and configures an HTTP server from the provided configuration.
func NewServer(cfg config.Config, handler http.Handler) *http.Server {
	addr := cfg.Port
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}

// Start starts the HTTP server in a goroutine and logs the listening address.
func (a *App) Start() error {
	go func() {
		a.logger.Info("server listening", zap.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Fatal("server error", zap.Error(err))
		}
	}()
	return nil
}

// Server returns the HTTP server instance for shutdown handling.
func (a *App) Server() *http.Server {
	return a.server
}

// resolveProjectPath locates a file or directory relative to the project root by walking up the directory tree.
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
