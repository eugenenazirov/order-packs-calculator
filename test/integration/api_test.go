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

func startServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := storage.NewMemoryStorage()
	calc := calculator.New()
	handler := api.NewHandler(calc, store)
	logger := zaptest.NewLogger(t)
	router := api.NewRouter(handler, logger)

	return httptest.NewServer(router)
}

func TestIntegrationFlow(t *testing.T) {
	server := startServer(t)
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from health, got %d", resp.StatusCode)
	}

	updatePayload := map[string]any{"packSizes": []int{23, 31, 53}}
	payload, _ := json.Marshal(updatePayload)
	req, err := http.NewRequest(http.MethodPut, server.URL+"/api/pack-sizes", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("pack sizes update failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from pack sizes update, got %d", resp.StatusCode)
	}

	calcPayload := map[string]any{"items": 500_000}
	body, _ := json.Marshal(calcPayload)
	resp, err = http.Post(server.URL+"/api/calculate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("calculate request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from calculate, got %d", resp.StatusCode)
	}
	var response struct {
		TotalItems int `json:"totalItems"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.TotalItems != 500_000 {
		t.Fatalf("unexpected total items %d", response.TotalItems)
	}
}
