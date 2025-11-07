package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestLoggingMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	var called bool
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
