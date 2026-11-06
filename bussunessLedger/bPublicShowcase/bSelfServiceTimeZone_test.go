package blpublicshowcase

import (
	"testing"
	"time"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// "Until 2 hours before" must use the company's clock (2026-11-04).
// 10 July 2026, 11:30 UTC = 07:30 in New York (GMT-4, summer time).
// An appointment at 09:00 New York is 1.5 hours away: no more changes.
// With the old Costa Rica / server clock, 09:00 would be 15:00 UTC (3.5 h).
func TestAppointmentChangeWindowUsesCompanyZone(t *testing.T) {
	now := time.Date(2026, 7, 10, 11, 30, 0, 0, time.UTC)
	restore := bltimezone.OverrideForTests(
		func(string) (string, *string, error) { return "America/New_York", nil, nil },
		func() time.Time { return now },
	)
	defer restore()

	a := &appointment.AppointmentS{CompanyID: "ny-company", AppointmentDate: "2026-07-10", StartHour: "09:00", Status: "SCHEDULED"}
	if appointmentCanChange(a) {
		t.Fatal("1.5 hours before the appointment the customer must not be able to change it")
	}
	if !appointmentCanRequestCancel(a) {
		t.Fatal("before the start the customer can still request a cancellation")
	}

	later := &appointment.AppointmentS{CompanyID: "ny-company", AppointmentDate: "2026-07-10", StartHour: "10:00", Status: "SCHEDULED"}
	if !appointmentCanChange(later) {
		t.Fatal("2.5 hours before the appointment the customer can change it")
	}

	start, ok := appointmentStart(a)
	if !ok || start.UTC().Format("15:04") != "13:00" {
		t.Fatalf("09:00 New York in July = %s UTC, want 13:00", start.UTC().Format("15:04"))
	}
}
