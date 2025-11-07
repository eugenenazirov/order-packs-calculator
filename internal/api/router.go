package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// RouterOption configures the behaviour of NewRouter.
type RouterOption func(*routerConfig)

// WithLogging controls whether access logs are emitted.
func WithLogging(enabled bool) RouterOption {
	return func(cfg *routerConfig) {
		cfg.enableLogging = enabled
	}
}

// WithRateLimiter overrides the default request rate limiter (primarily for tests).
func WithRateLimiter(limiter rateLimiter) RouterOption {
	return func(cfg *routerConfig) {
		cfg.rateLimiter = limiter
	}
}

type routerConfig struct {
	enableLogging bool
	logger        *zap.Logger
	rateLimiter   rateLimiter
}

// NewRouter creates an HTTP router with standard middleware.
func NewRouter(handler *Handler, logger *zap.Logger, opts ...RouterOption) http.Handler {
	cfg := routerConfig{
		enableLogging: true,
		logger:        logger,
		rateLimiter:   newTokenBucketLimiter(25, 50),
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
	root = recoveryMiddleware(cfg.logger, root)
	if cfg.enableLogging {
		root = loggingMiddleware(cfg.logger, root)
	}
	root = rateLimitMiddleware(cfg.rateLimiter, root)
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

func loggingMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		requestID := requestIDFromContext(r.Context())
		logger.Info("request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
			zap.Duration("duration", duration),
			zap.String("request_id", requestID),
		)
	})
}

func recoveryMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", zap.Any("error", rec))
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
