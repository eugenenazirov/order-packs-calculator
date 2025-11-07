package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

type controllableClock struct {
	mu  sync.RWMutex
	now time.Time
}

func newControllableClock(initial time.Time) *controllableClock {
	return &controllableClock{now: initial}
}

func (c *controllableClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *controllableClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func setupTestRouter(t *testing.T) (http.Handler, *controllableClock) {
	t.Helper()

	store := storage.NewMemoryStorage()
	calc := calculator.New()
	clock := newControllableClock(time.Date(2024, 11, 1, 12, 0, 0, 0, time.UTC))

	handler := NewHandler(calc, store, WithClock(clock.Now))
	logger := zaptest.NewLogger(t)
	router := NewRouter(handler, logger, WithLogging(false))

	return router, clock
}

func TestRequestIDHelpers(t *testing.T) {
	ctx := contextWithRequestID(context.Background(), "abc")
	if got := requestIDFromContext(ctx); got != "abc" {
		t.Fatalf("expected abc, got %s", got)
	}
	resp := httptest.NewRecorder()
	writeInternalError(resp, assertError("boom"))
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 status, got %d", resp.Code)
	}
}

type assertError string

func (a assertError) Error() string { return string(a) }

func TestHealthEndpoint(t *testing.T) {
	router, clock := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %s", body.Status)
	}
	if !body.Timestamp.Equal(clock.Now()) {
		t.Fatalf("expected timestamp %s, got %s", clock.Now(), body.Timestamp)
	}
}

func TestGetPackSizesReturnsDefaults(t *testing.T) {
	router, clock := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		PackSizes []int     `json:"packSizes"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	want := storage.DefaultPackSizes()
	if len(body.PackSizes) != len(want) {
		t.Fatalf("expected %d pack sizes, got %d", len(want), len(body.PackSizes))
	}
	for i, size := range want {
		if body.PackSizes[i] != size {
			t.Fatalf("expected pack size %d at position %d, got %d", size, i, body.PackSizes[i])
		}
	}
	if !body.UpdatedAt.Equal(clock.Now()) {
		t.Fatalf("expected updatedAt %s, got %s", clock.Now(), body.UpdatedAt)
	}
}

func TestPutPackSizesUpdatesStorage(t *testing.T) {
	router, clock := setupTestRouter(t)

	clock.Advance(time.Hour)

	payload := map[string]any{
		"packSizes": []int{53, 23, 31},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		PackSizes []int     `json:"packSizes"`
		UpdatedAt time.Time `json:"updatedAt"`
		Message   string    `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Message == "" {
		t.Fatalf("expected success message, got empty string")
	}

	want := []int{23, 31, 53}
	if len(body.PackSizes) != len(want) {
		t.Fatalf("expected %d pack sizes, got %d", len(want), len(body.PackSizes))
	}
	for i, size := range want {
		if body.PackSizes[i] != size {
			t.Fatalf("expected pack size %d at position %d, got %d", size, i, body.PackSizes[i])
		}
	}

	if !body.UpdatedAt.Equal(clock.Now()) {
		t.Fatalf("expected updatedAt %s, got %s", clock.Now(), body.UpdatedAt)
	}
}

func TestPutPackSizesValidatesInput(t *testing.T) {
	router, _ := setupTestRouter(t)

	payload := map[string]any{
		"packSizes": []int{},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCalculateEndpointSuccess(t *testing.T) {
	router, clock := setupTestRouter(t)

	clock.Advance(time.Minute)

	payload := map[string]any{
		"items": 750,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Items      int            `json:"items"`
		Packs      map[string]int `json:"packs"`
		TotalPacks int            `json:"totalPacks"`
		TotalItems int            `json:"totalItems"`
		Remainder  int            `json:"remainder"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Items != 750 {
		t.Fatalf("expected items 750, got %d", body.Items)
	}
	if body.TotalPacks != 2 {
		t.Fatalf("expected total packs 2, got %d", body.TotalPacks)
	}
	if body.TotalItems != 750 {
		t.Fatalf("expected total items 750, got %d", body.TotalItems)
	}
	if body.Remainder != 0 {
		t.Fatalf("expected remainder 0, got %d", body.Remainder)
	}
}

func TestCalculateEndpointRejectsZeroItems(t *testing.T) {
	router, _ := setupTestRouter(t)

	payload := map[string]any{
		"items": 0,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for zero items, got %d", rec.Code)
	}
}

func TestCalculateEndpointImpossible(t *testing.T) {
	router, _ := setupTestRouter(t)

	payload := map[string]any{
		"items": 263,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}

	var body struct {
		Suggestion string `json:"suggestion"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Suggestion == "" {
		t.Fatalf("expected suggestion to be populated")
	}
}

func TestCalculateEndpointEdgeCase(t *testing.T) {
	router, clock := setupTestRouter(t)

	clock.Advance(time.Minute)
	updatePayload := map[string]any{
		"packSizes": []int{23, 31, 53},
	}
	updateData, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewReader(updateData))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for pack sizes update, got %d", updateRec.Code)
	}

	clock.Advance(time.Minute)

	payload := map[string]any{
		"items": 500_000,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Packs      map[string]int `json:"packs"`
		TotalPacks int            `json:"totalPacks"`
		TotalItems int            `json:"totalItems"`
		Remainder  int            `json:"remainder"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expected := map[string]int{
		"23": 2,
		"31": 7,
		"53": 9429,
	}
	for pack, want := range expected {
		got, ok := body.Packs[pack]
		if !ok {
			t.Fatalf("expected pack %s to be present", pack)
		}
		if got != want {
			t.Fatalf("expected pack %s count %d, got %d", pack, want, got)
		}
	}
	if body.TotalPacks != 9438 {
		t.Fatalf("expected total packs 9438, got %d", body.TotalPacks)
	}
	if body.TotalItems != 500_000 {
		t.Fatalf("expected total items 500000, got %d", body.TotalItems)
	}
	if body.Remainder != 0 {
		t.Fatalf("expected remainder 0, got %d", body.Remainder)
	}
}

func TestCorsPreflight(t *testing.T) {
	router, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/calculate", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be set")
	}
}

func TestRequestIDPropagation(t *testing.T) {
	router, _ := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("X-Request-ID", "test-request-id")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "test-request-id" {
		t.Fatalf("expected X-Request-ID header to be echoed, got %s", got)
	}
}
