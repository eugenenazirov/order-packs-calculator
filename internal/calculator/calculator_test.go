package calculator

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestCalculatePacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		items     int
		packSizes []int
		want      map[int]int
		wantErr   error
	}{
		{
			name:      "SimpleCombination",
			items:     750,
			packSizes: []int{250, 500, 1000},
			want: map[int]int{
				250: 1,
				500: 1,
			},
		},
		{
			name:      "ExactMatchSinglePack",
			items:     1000,
			packSizes: []int{250, 500, 1000},
			want: map[int]int{
				1000: 1,
			},
		},
		{
			name:      "SinglePackSize",
			items:     1000,
			packSizes: []int{100},
			want: map[int]int{
				100: 10,
			},
		},
		{
			name:      "EdgeCaseLargeNumber",
			items:     500_000,
			packSizes: []int{23, 31, 53},
			want: map[int]int{
				23: 2,
				31: 7,
				53: 9429,
			},
		},
		{
			name:      "CoprimePackSizes",
			items:     100,
			packSizes: []int{7, 13},
			want: map[int]int{
				7:  5,
				13: 5,
			},
		},
		{
			name:      "ZeroItems",
			items:     0,
			packSizes: []int{250, 500},
			want:      map[int]int{},
		},
		{
			name:      "ItemsLessThanSmallestPack",
			items:     100,
			packSizes: []int{250, 500},
			wantErr:   ErrCannotFulfill,
		},
		{
			name:      "SimpleCase263ItemsCannotFulfill",
			items:     263,
			packSizes: []int{250, 500, 1000},
			wantErr:   ErrCannotFulfill,
		},
		{
			name:      "ImpossibleToPackExactly",
			items:     7,
			packSizes: []int{3, 5},
			wantErr:   ErrCannotFulfill,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := New().CalculatePacks(tc.items, tc.packSizes)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				return
			}

			if !equalDistributions(got, tc.want) {
				t.Fatalf("unexpected result: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCalculatePacks_InvalidItems(t *testing.T) {
	t.Parallel()

	if _, err := New().CalculatePacks(-1, []int{1}); !errors.Is(err, ErrInvalidItems) {
		t.Fatalf("expected ErrInvalidItems, got %v", err)
	}
}

func TestCalculatePacks_InvalidPackSizes(t *testing.T) {
	t.Parallel()

	invalidCases := [][]int{
		nil,
		{},
		{0, 10},
		{-5, 10},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
	}

	for _, packSizes := range invalidCases {
		packSizes := packSizes
		t.Run(fmt.Sprintf("%v", packSizes), func(t *testing.T) {
			if _, err := New().CalculatePacks(10, packSizes); !errors.Is(err, ErrInvalidPackSizes) {
				t.Fatalf("expected ErrInvalidPackSizes for %v, got %v", packSizes, err)
			}
		})
	}
}

func TestNormalizePackSizes_SortsAndDeduplicates(t *testing.T) {
	t.Parallel()

	input := []int{53, 23, 31, 23, 53}
	got, err := normalizePackSizes(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{23, 31, 53}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func equalDistributions(got, want map[int]int) bool {
	if len(got) != len(want) {
		return false
	}
	for k, wantVal := range want {
		if gotVal := got[k]; gotVal != wantVal {
			return false
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			return false
		}
	}
	return true
}

func BenchmarkCalculatePacksSmall(b *testing.B) {
	calc := New()
	packSizes := []int{3, 5, 9}
	for i := 0; i < b.N; i++ {
		if _, err := calc.CalculatePacks(999, packSizes); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkCalculatePacksMedium(b *testing.B) {
	calc := New()
	packSizes := []int{23, 31, 53}
	for i := 0; i < b.N; i++ {
		if _, err := calc.CalculatePacks(50_000, packSizes); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkCalculatePacksLarge(b *testing.B) {
	calc := New()
	packSizes := []int{23, 31, 53}
	for i := 0; i < b.N; i++ {
		if _, err := calc.CalculatePacks(500_000, packSizes); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
