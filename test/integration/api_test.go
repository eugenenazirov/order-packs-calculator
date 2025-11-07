package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/eugenenazirov/re-partners/internal/api"
	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

func newRouter(t *testing.T) http.Handler {
	t.Helper()

	store := storage.NewMemoryStorage()
	calc := calculator.New()
	handler := api.NewHandler(calc, store)
	logger := zaptest.NewLogger(t)
	return api.NewRouter(handler, logger)
}

func performRequest(t *testing.T, handler http.Handler, method, target string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestIntegrationFlow(t *testing.T) {
	handler := newRouter(t)

	rec := performRequest(t, handler, http.MethodGet, "/api/health", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from health, got %d", rec.Code)
	}

	updatePayload := map[string]any{"packSizes": []int{23, 31, 53}}
	payload, _ := json.Marshal(updatePayload)
	rec = performRequest(t, handler, http.MethodPut, "/api/pack-sizes", payload, map[string]string{"Content-Type": "application/json"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from pack sizes update, got %d", rec.Code)
	}

	calcPayload := map[string]any{"items": 500_000}
	body, _ := json.Marshal(calcPayload)
	rec = performRequest(t, handler, http.MethodPost, "/api/calculate", body, map[string]string{"Content-Type": "application/json"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from calculate, got %d", rec.Code)
	}

	var response struct {
		TotalItems int `json:"totalItems"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TotalItems != 500_000 {
		t.Fatalf("unexpected total items %d", response.TotalItems)
	}
}
