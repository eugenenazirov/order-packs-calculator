package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RouterOption configures the behaviour of NewRouter.
type RouterOption func(*routerConfig)

// WithLogging controls whether access logs are emitted.
func WithLogging(enabled bool) RouterOption {
	return func(cfg *routerConfig) {
		cfg.enableLogging = enabled
	}
}

type routerConfig struct {
	enableLogging bool
}

// NewRouter creates an HTTP router with standard middleware.
func NewRouter(handler *Handler, opts ...RouterOption) http.Handler {
	cfg := routerConfig{
		enableLogging: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", http.HandlerFunc(handler.handleHealth))
	mux.Handle("GET /api/pack-sizes", http.HandlerFunc(handler.handleGetPackSizes))
	mux.Handle("PUT /api/pack-sizes", http.HandlerFunc(handler.handlePutPackSizes))
	mux.Handle("POST /api/calculate", http.HandlerFunc(handler.handleCalculate))

	var root http.Handler = mux
	root = corsMiddleware(root)
	root = recoveryMiddleware(root)
	if cfg.enableLogging {
		root = loggingMiddleware(root)
	}
	root = requestIDMiddleware(root)

	return root
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		requestID := requestIDFromContext(r.Context())
		log.Printf("%s %s %d %s request_id=%s", r.Method, r.URL.Path, rec.status, duration, requestID)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				writeError(w, http.StatusInternalServerError, "Internal error", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = generateRequestID()
		}
		ctx := r.Context()
		ctx = contextWithRequestID(ctx, requestID)

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf)
}

func contextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, id)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
