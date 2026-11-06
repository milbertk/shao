package blorder

import "testing"

func TestIsAppointmentBehavior(t *testing.T) {
	if !IsAppointmentBehavior(" booking ") || IsAppointmentBehavior("MESSENGER_APP") || IsAppointmentBehavior("") {
		t.Fatal("appointment rule")
	}
}
