package blnotifications

import (
	"testing"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
)

// Appointment hours in emails show the company zone ON THAT DATE (2026-11-04).
func TestTimeRangeOnShowsCompanyZone(t *testing.T) {
	cr := companyContext{Language: "es", Location: bltimezone.LocationFor("America/Costa_Rica"), ZoneLabel: "Hora de Costa Rica"}
	if got := timeRangeOn(cr, "2026-11-20", "10:00:00", "11:00:00"); got != "10:00–11:00 · Hora de Costa Rica, GMT−6" {
		t.Fatalf("Costa Rica: %q", got)
	}
	ny := companyContext{Language: "en", Location: bltimezone.LocationFor("America/New_York")}
	if got := timeRangeOn(ny, "2026-07-10", "09:00", "10:00"); got != "09:00–10:00 · New York, GMT−4" {
		t.Fatalf("New York summer: %q", got)
	}
	if got := timeRangeOn(ny, "2026-01-10", "09:00", "10:00"); got != "09:00–10:00 · New York, GMT−5" {
		t.Fatalf("New York winter: %q", got)
	}
	// No zone (old callers / tests): unchanged text.
	if got := timeRangeOn(companyContext{Language: "en"}, "2026-01-10", "09:00", "10:00"); got != "09:00–10:00" {
		t.Fatalf("no zone: %q", got)
	}
}
