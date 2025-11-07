package storage

import (
	"errors"
	"sort"
	"sync"
)

const maxPackSizes = 10

var (
	// ErrInvalidPackSizes indicates the provided pack sizes violate validation rules.
	ErrInvalidPackSizes = errors.New("pack sizes must contain between 1 and 10 positive integers")
)

var defaultPackSizes = []int{250, 500, 1000, 2000, 5000}

// Storage provides access to the pack sizes used by the calculator.
type Storage interface {
	GetPackSizes() ([]int, error)
	SetPackSizes(sizes []int) error
}

// MemoryStorage keeps pack sizes in-memory and guards access with a RWMutex.
type MemoryStorage struct {
	mu        sync.RWMutex
	packSizes []int
}

// NewMemoryStorage initialises storage with a copy of the default pack sizes.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		packSizes: cloneAndSort(defaultPackSizes),
	}
}

// DefaultPackSizes returns a copy of the default pack sizes slice.
func DefaultPackSizes() []int {
	return cloneAndSort(defaultPackSizes)
}

// GetPackSizes returns a defensive copy of the currently configured pack sizes.
func (s *MemoryStorage) GetPackSizes() ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneAndSort(s.packSizes), nil
}

// SetPackSizes validates, normalises, and stores the provided pack sizes.
func (s *MemoryStorage) SetPackSizes(sizes []int) error {
	normalized, err := normalizePackSizes(sizes)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.packSizes = normalized
	s.mu.Unlock()

	return nil
}

func cloneAndSort(src []int) []int {
	if len(src) == 0 {
		return []int{}
	}

	out := make([]int, len(src))
	copy(out, src)
	sort.Ints(out)
	return out
}

func normalizePackSizes(packSizes []int) ([]int, error) {
	if len(packSizes) == 0 {
		return nil, ErrInvalidPackSizes
	}

	unique := make(map[int]struct{}, len(packSizes))
	for _, size := range packSizes {
		if size <= 0 {
			return nil, ErrInvalidPackSizes
		}
		unique[size] = struct{}{}
		if len(unique) > maxPackSizes {
			return nil, ErrInvalidPackSizes
		}
	}

	out := make([]int, 0, len(unique))
	for size := range unique {
		out = append(out, size)
	}
	sort.Ints(out)
	return out, nil
}
