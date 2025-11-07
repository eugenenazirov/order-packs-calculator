package calculator

import (
	"sort"
)

const maxPackSizes = 10

type dpCalculator struct{}

// New creates a Calculator based on dynamic programming.
func New() Calculator {
	return &dpCalculator{}
}

func (c *dpCalculator) CalculatePacks(items int, packSizes []int) (map[int]int, error) {
	if items < 0 {
		return nil, ErrInvalidItems
	}
	normalized, err := normalizePackSizes(packSizes)
	if err != nil {
		return nil, err
	}
	if items == 0 {
		return map[int]int{}, nil
	}
	if items < normalized[0] {
		return nil, ErrCannotFulfill
	}

	dp := make([]int, items+1)
	choice := make([]int, items+1)
	inf := items + 1

	for i := 1; i <= items; i++ {
		dp[i] = inf
		choice[i] = -1
	}

	for _, size := range normalized {
		for amount := size; amount <= items; amount++ {
			prev := amount - size
			if dp[prev]+1 < dp[amount] {
				dp[amount] = dp[prev] + 1
				choice[amount] = size
			}
		}
	}

	if choice[items] == -1 {
		return nil, ErrCannotFulfill
	}

	result := make(map[int]int, len(normalized))
	for remaining := items; remaining > 0; {
		size := choice[remaining]
		if size <= 0 {
			return nil, ErrCannotFulfill
		}
		result[size]++
		remaining -= size
	}

	return result, nil
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

	normalized := make([]int, 0, len(unique))
	for size := range unique {
		normalized = append(normalized, size)
	}
	sort.Ints(normalized)

	return normalized, nil
}
