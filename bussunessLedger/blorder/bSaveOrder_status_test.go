package blorder

import "testing"

func TestResolveOrderStatus(t *testing.T) {
	cases := []struct {
		create  bool
		current string
		want    string
	}{
		{true, "", "PLACED"},
		{true, "SCHEDULED", "PLACED"},
		{false, "in_progress", "IN_PROGRESS"},
		{false, "PARTIAL_DONE", "PARTIAL_DONE"},
		{false, "", "PLACED"},
	}
	for _, c := range cases {
		if got := resolveOrderStatus(c.create, c.current); got != c.want {
			t.Errorf("resolveOrderStatus(%v, %q) = %q, want %q", c.create, c.current, got, c.want)
		}
	}
}
