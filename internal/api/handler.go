package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/eugenenazirov/re-partners/internal/calculator"
	"github.com/eugenenazirov/re-partners/internal/storage"
)

type contextKey string

const requestIDContextKey contextKey = "requestID"

// Handler wires calculator and storage dependencies into HTTP handlers.
type Handler struct {
	calculator calculator.Calculator
	storage    storage.Storage

	clock func() time.Time

	mu                 sync.RWMutex
	packSizesUpdatedAt time.Time
}

// HandlerOption configures Handler behaviour.
type HandlerOption func(*Handler)

// WithClock overrides the time source, primarily for tests.
func WithClock(clock func() time.Time) HandlerOption {
	return func(h *Handler) {
		h.clock = clock
	}
}

// NewHandler constructs a Handler with the provided dependencies.
func NewHandler(calc calculator.Calculator, store storage.Storage, opts ...HandlerOption) *Handler {
	h := &Handler{
		calculator: calc,
		storage:    store,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, opt := range opts {
		opt(h)
	}
	h.packSizesUpdatedAt = h.clock()
	return h
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = r
	resp := healthResponse{
		Status:    "ok",
		Timestamp: h.clock(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleGetPackSizes(w http.ResponseWriter, r *http.Request) {
	_ = r
	sizes, err := h.storage.GetPackSizes()
	if err != nil {
		writeInternalError(w, err)
		return
	}

	resp := packSizesResponse{
		PackSizes: sizes,
		UpdatedAt: h.currentPackSizesUpdatedAt(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handlePutPackSizes(w http.ResponseWriter, r *http.Request) {
	var req packSizesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request", "unable to parse JSON payload")
		return
	}

	if len(req.PackSizes) == 0 {
		writeError(w, http.StatusBadRequest, "Invalid pack sizes", "packSizes must contain at least one size")
		return
	}

	if err := h.storage.SetPackSizes(req.PackSizes); err != nil {
		if errors.Is(err, storage.ErrInvalidPackSizes) {
			writeError(w, http.StatusBadRequest, "Invalid pack sizes", err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}

	h.markPackSizesUpdated()

	sizes, err := h.storage.GetPackSizes()
	if err != nil {
		writeInternalError(w, err)
		return
	}

	resp := packSizesResponse{
		PackSizes: sizes,
		UpdatedAt: h.currentPackSizesUpdatedAt(),
		Message:   "Pack sizes updated successfully",
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleCalculate(w http.ResponseWriter, r *http.Request) {
	var req calculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request", "unable to parse JSON payload")
		return
	}

	if req.Items <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid request", "items must be a positive integer")
		return
	}

	packSizes, err := h.storage.GetPackSizes()
	if err != nil {
		writeInternalError(w, err)
		return
	}

	start := time.Now()
	result, calcErr := h.calculator.CalculatePacks(req.Items, packSizes)
	elapsed := time.Since(start)

	if calcErr != nil {
		switch {
		case errors.Is(calcErr, calculator.ErrInvalidItems):
			writeError(w, http.StatusBadRequest, "Invalid request", calcErr.Error())
		case errors.Is(calcErr, calculator.ErrCannotFulfill):
			suggestion := fmt.Sprintf("Consider adding a pack size that divides %d or adjust the order quantity", req.Items)
			writeError(w, http.StatusUnprocessableEntity, "Cannot pack exactly", calcErr.Error(), suggestion)
		case errors.Is(calcErr, calculator.ErrInvalidPackSizes):
			writeError(w, http.StatusInternalServerError, "Internal error", calcErr.Error())
		default:
			writeInternalError(w, calcErr)
		}
		return
	}

	packs := make(map[string]int, len(result))
	sizes := make([]int, 0, len(result))
	for size := range result {
		sizes = append(sizes, size)
	}
	sort.Ints(sizes)

	totalItems := 0
	totalPacks := 0
	for _, size := range sizes {
		count := result[size]
		packs[strconv.Itoa(size)] = count
		totalItems += size * count
		totalPacks += count
	}

	resp := calculateResponse{
		Items:             req.Items,
		Packs:             packs,
		TotalPacks:        totalPacks,
		TotalItems:        totalItems,
		Remainder:         req.Items - totalItems,
		CalculationTimeMs: elapsed.Milliseconds(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) currentPackSizesUpdatedAt() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.packSizesUpdatedAt
}

func (h *Handler) markPackSizesUpdated() {
	h.mu.Lock()
	h.packSizesUpdatedAt = h.clock()
	h.mu.Unlock()
}

func requestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(requestIDContextKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

type packSizesRequest struct {
	PackSizes []int `json:"packSizes"`
}

type calculateRequest struct {
	Items int `json:"items"`
}

type calculateResponse struct {
	Items             int            `json:"items"`
	Packs             map[string]int `json:"packs"`
	TotalPacks        int            `json:"totalPacks"`
	TotalItems        int            `json:"totalItems"`
	Remainder         int            `json:"remainder"`
	CalculationTimeMs int64          `json:"calculationTimeMs"`
}

type packSizesResponse struct {
	PackSizes []int     `json:"packSizes"`
	UpdatedAt time.Time `json:"updatedAt"`
	Message   string    `json:"message,omitempty"`
}

type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type errorResponse struct {
	Error      string `json:"error"`
	Details    string `json:"details,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message, details string, suggestion ...string) {
	resp := errorResponse{
		Error:   message,
		Details: details,
	}
	if len(suggestion) > 0 {
		resp.Suggestion = suggestion[0]
	}
	writeJSON(w, status, resp)
}

func writeInternalError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, "Internal error", err.Error())
}
