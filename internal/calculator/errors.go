package calculator

import "errors"

var (
	// ErrInvalidItems is returned when the requested number of items is negative.
	ErrInvalidItems = errors.New("items must be a non-negative integer")
	// ErrInvalidPackSizes is returned when pack sizes are missing or contain invalid entries.
	ErrInvalidPackSizes = errors.New("pack sizes must contain between 1 and 10 positive integers")
	// ErrCannotFulfill is returned when it is impossible to pack the items exactly with the provided sizes.
	ErrCannotFulfill = errors.New("cannot pack items exactly with the provided pack sizes")
)
