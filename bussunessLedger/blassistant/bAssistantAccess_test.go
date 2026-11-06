package blassistant

import (
	"testing"
	"time"
)

func TestTokensCostUSD(t *testing.T) {
	// 2M input at $1 + 400k output at $5 = 2 + 2 = 4
	if got := tokensCostUSD(2_000_000, 400_000, 1, 5); got != 4 {
		t.Fatalf("cost = %v, want 4", got)
	}
	if got := tokensCostUSD(0, 0, 1, 5); got != 0 {
		t.Fatalf("cost = %v, want 0", got)
	}
}

func TestMonthStart(t *testing.T) {
	now := time.Date(2026, 10, 22, 17, 40, 0, 0, time.UTC)
	if got := monthStart(now); !got.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthStart = %v", got)
	}
}
