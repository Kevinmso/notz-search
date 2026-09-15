package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{ErrEmbeddingGeneration, ErrUpsert, ErrSearch, ErrInvalidPayload}

	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel %d should not match sentinel %d", i, j)
			}
		}
	}
}

func TestSentinelErrors_WrappedWithFmtErrorf(t *testing.T) {
	cause := errors.New("boom")
	wrapped := fmt.Errorf("%w: %w", ErrUpsert, cause)

	if !errors.Is(wrapped, ErrUpsert) {
		t.Error("wrapped error should match ErrUpsert")
	}
	if !errors.Is(wrapped, cause) {
		t.Error("wrapped error should match the original cause")
	}
	if errors.Is(wrapped, ErrSearch) {
		t.Error("wrapped error should not match an unrelated sentinel")
	}
}
