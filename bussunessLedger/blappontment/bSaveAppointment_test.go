package blappointment

import (
	"testing"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
)

func TestAppointmentUpdateEvent(t *testing.T) {
	base := blnotifications.AppointmentSnapshot{ServiceID: "s1", Date: "2026-11-02", StartHour: "10:00", EndHour: "11:00", Status: "SCHEDULED"}
	cases := []struct {
		name   string
		change func(*blnotifications.AppointmentSnapshot)
		want   string
	}{
		{"nothing", func(a *blnotifications.AppointmentSnapshot) {}, ""},
		{"empty status keeps old", func(a *blnotifications.AppointmentSnapshot) { a.Status = "" }, ""},
		{"new time", func(a *blnotifications.AppointmentSnapshot) { a.StartHour = "14:00"; a.EndHour = "15:00" }, blnotifications.EventModified},
		{"new service", func(a *blnotifications.AppointmentSnapshot) { a.ServiceID = "s2" }, blnotifications.EventModified},
		{"done", func(a *blnotifications.AppointmentSnapshot) { a.Status = "DONE" }, blnotifications.EventDone},
		{"cancelled + moved", func(a *blnotifications.AppointmentSnapshot) { a.Status = "CANCELLED"; a.Date = "2026-11-03" }, blnotifications.EventCancelled},
	}
	for _, c := range cases {
		after := base
		c.change(&after)
		if got := appointmentUpdateEvent(base, after); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
	pending := base
	pending.Status = "PENDING"
	if got := appointmentUpdateEvent(pending, base); got != blnotifications.EventApproved {
		t.Errorf("pending -> scheduled: got %q", got)
	}
	cancelled := base
	cancelled.Status = "CANCELLED"
	moved := cancelled
	moved.Date = "2026-12-01"
	if got := appointmentUpdateEvent(cancelled, moved); got != "" {
		t.Errorf("moving a cancelled appointment must not email: %q", got)
	}
}
