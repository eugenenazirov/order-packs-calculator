package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type staticLimiter struct {
	allow bool
}

func (s *staticLimiter) Allow() bool {
	return s.allow
}

func TestRateLimitMiddlewareBlocksWhenLimiterDenies(t *testing.T) {
	middleware := rateLimitMiddleware(&staticLimiter{allow: false}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatalf("handler should not execute when rate limited")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddlewarePassesWhenLimiterAllows(t *testing.T) {
	var called bool
	middleware := rateLimitMiddleware(&staticLimiter{allow: true}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	middleware.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected handler to execute when limiter allows")
	}
}

func TestNewTokenBucketLimiterUsesDefaults(t *testing.T) {
	limiter := newTokenBucketLimiter(0, 0)
	if limiter == nil {
		t.Fatalf("expected limiter instance")
	}
	if !limiter.Allow() {
		t.Fatalf("expected first request to be allowed")
	}
}
