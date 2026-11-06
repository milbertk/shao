package blnotifications

import "testing"

func TestFirstEmail(t *testing.T) {
	calls := 0
	counted := func(value string) func() string {
		return func() string { calls++; return value }
	}
	if got := firstEmail(fixed(""), fixed("not-an-email"), counted("ana@example.com"), counted("other@example.com")); got != "ana@example.com" {
		t.Fatalf("got %q", got)
	}
	if calls != 1 {
		t.Fatalf("lookups after the first valid email must not run, calls=%d", calls)
	}
	if got := firstEmail(fixed(""), fixed("  ")); got != "" {
		t.Fatalf("no email must give empty, got %q", got)
	}
}
