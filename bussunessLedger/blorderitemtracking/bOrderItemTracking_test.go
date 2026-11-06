package blorderitemtracking

import "testing"

func TestItemStepEmail(t *testing.T) {
	cases := []struct {
		prev, next      string
		changed, notify bool
		want            string
	}{
		{"PLACED", "IN_PROGRESS", true, false, emailOrderStatus},  // unchecked step, status changed
		{"IN_PROGRESS", "IN_PROGRESS", true, false, emailNone},    // unchecked step, same status
		{"IN_PROGRESS", "IN_PROGRESS", true, true, emailItemStep}, // checked step
		{"PARTIAL_DONE", "DONE", true, true, emailItemStep},       // checked step shows the status too
		{"PARTIAL_DONE", "DONE", true, false, emailOrderStatus},   // order finished: always
		{"CANCELLED", "CANCELLED", true, true, emailNone},
		{"", "IN_PROGRESS", false, false, emailNone},
	}
	for _, c := range cases {
		if got := itemStepEmail(c.prev, c.next, c.changed, c.notify); got != c.want {
			t.Errorf("%+v: got %q", c, got)
		}
	}
}
