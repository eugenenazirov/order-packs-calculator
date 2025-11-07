package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"slices"
)

func TestNewMemoryStorageReturnsDefaultSizes(t *testing.T) {
	t.Parallel()

	store := NewMemoryStorage()

	got, err := store.GetPackSizes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := DefaultPackSizes()
	if !slices.Equal(got, want) {
		t.Fatalf("expected default sizes %v, got %v", want, got)
	}

	// ensure mutation safety
	got[0] = 999
	again, err := store.GetPackSizes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slices.Equal(again, got) {
		t.Fatalf("expected defensive copy, got %v", again)
	}
}

func TestSetPackSizesUpdatesState(t *testing.T) {
	t.Parallel()

	store := NewMemoryStorage()
	if err := store.SetPackSizes([]int{1000, 250, 500, 500}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetPackSizes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{250, 500, 1000}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSetPackSizesRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	testCases := [][]int{
		nil,
		{},
		{0, 10},
		{-5, 100},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
	}

	for idx, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("case_%d", idx), func(t *testing.T) {
			store := NewMemoryStorage()
			if err := store.SetPackSizes(tc); !errors.Is(err, ErrInvalidPackSizes) {
				t.Fatalf("expected ErrInvalidPackSizes for %v, got %v", tc, err)
			}
		})
	}
}

func TestMemoryStorageConcurrentAccess(t *testing.T) {
	store := NewMemoryStorage()
	var wg sync.WaitGroup

	for i := 0; i < 32; i++ {
		wg.Add(2)

		go func(offset int) {
			defer wg.Done()
			sizes := []int{250 + offset, 500 + offset}
			if err := store.SetPackSizes(sizes); err != nil {
				t.Errorf("SetPackSizes failed: %v", err)
			}
		}(i)

		go func() {
			defer wg.Done()
			if _, err := store.GetPackSizes(); err != nil {
				t.Errorf("GetPackSizes failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// final read should succeed
	if _, err := store.GetPackSizes(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
