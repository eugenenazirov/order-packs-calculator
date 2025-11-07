package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildRootHandler(t *testing.T) {
	apiInvoked := false
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Fatalf("unexpected path passed to API handler: %s", r.URL.Path)
		}
		apiInvoked = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler, err := buildRootHandler(apiHandler)
	if err != nil {
		t.Fatalf("buildRootHandler returned error: %v", err)
	}

	t.Run("serves index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if rec.Header().Get("Content-Type") == "" {
			t.Fatalf("expected Content-Type header for index page")
		}
	})

	t.Run("returns not found for unknown paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("forwards api traffic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d", rec.Code)
		}
		if !apiInvoked {
			t.Fatalf("expected API handler to be invoked")
		}
	})
}
