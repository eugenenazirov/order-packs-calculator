package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/storage"
	"go.uber.org/zap/zaptest"
)

func TestLoggingMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	var called bool
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected handler to be called")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := recoveryMiddleware(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(errors.New("boom"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", rec.Code)
	}
}

func TestResponseRecorderWriteHeader(t *testing.T) {
	underlying := httptest.NewRecorder()
	rec := &responseRecorder{ResponseWriter: underlying}
	rec.WriteHeader(http.StatusTeapot)

	if rec.status != http.StatusTeapot {
		t.Fatalf("expected status to be recorded")
	}
	if underlying.Code != http.StatusTeapot {
		t.Fatalf("expected status to propagate to ResponseWriter")
	}
}

func TestWithRateLimiterOptionAppliesLimiter(t *testing.T) {
	router := newTestRouter(t, WithLogging(false), WithRateLimiter(&staticLimiter{allow: false}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limiter to block request, got %d", rec.Code)
	}
}

func TestWithRateLimitDisablesLimiterWhenZero(t *testing.T) {
	router := newTestRouter(t, WithLogging(false), WithRateLimiter(&staticLimiter{allow: false}), WithRateLimit(0, 0))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected limiter to be disabled, got %d", rec.Code)
	}
}

func TestWithRateLimitEnforcesLimit(t *testing.T) {
	router := newTestRouter(t, WithLogging(false), WithRateLimit(1, 1))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req.Clone(req.Context()))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limiter to block second request, got %d", rec2.Code)
	}
}

func newTestRouter(t *testing.T, opts ...RouterOption) http.Handler {
	t.Helper()

	store := storage.NewMemoryStorage()
	calc := calculator.New()
	handler := NewHandler(calc, store)
	logger := zaptest.NewLogger(t)
	return NewRouter(handler, logger, opts...)
}
