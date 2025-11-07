package logging

import "testing"

func TestNew(t *testing.T) {
	logger, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatalf("expected logger instance")
	}
	_ = logger.Sync()
}
