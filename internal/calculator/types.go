package calculator

// PackResult represents a summary of the packing calculation.
// TotalPacks and TotalItems are derived values that callers can use when they
// need aggregated information in addition to the raw distribution.
type PackResult struct {
	Packs      map[int]int
	TotalItems int
	TotalPacks int
}

// Calculator describes the behaviour required from a pack calculator.
type Calculator interface {
	CalculatePacks(items int, packSizes []int) (map[int]int, error)
}
