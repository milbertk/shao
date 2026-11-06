package bltimezone

import (
	"errors"
	"testing"
	"time"
)

// useFakeZone makes CompanyZone return the given zone without a database.
func useFakeZone(t *testing.T, zone string, label *string, now time.Time) {
	t.Helper()
	oldNow, oldLoad := nowFunc, loadZoneFromDB
	nowFunc = func() time.Time { return now }
	loadZoneFromDB = func(string) (string, *string, error) { return zone, label, nil }
	cacheMu.Lock()
	cache = map[string]cachedZone{}
	cacheMu.Unlock()
	t.Cleanup(func() {
		nowFunc, loadZoneFromDB = oldNow, oldLoad
		cacheMu.Lock()
		cache = map[string]cachedZone{}
		cacheMu.Unlock()
	})
}

func TestNormalizeTimeZone(t *testing.T) {
	cases := map[string]string{
		"":                    DefaultTimeZone,
		"  America/New_York ": "America/New_York",
		"Europe/Madrid":       "Europe/Madrid",
		"UTC":                 "UTC",
	}
	for in, want := range cases {
		got, err := NormalizeTimeZone(in)
		if err != nil || got != want {
			t.Fatalf("NormalizeTimeZone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"Local", "GMT-6x", "Mars/Olympus", "../etc/passwd"} {
		if _, err := NormalizeTimeZone(bad); !errors.Is(err, ErrInvalidTimeZone) {
			t.Fatalf("NormalizeTimeZone(%q) should fail, got %v", bad, err)
		}
	}
}

func TestNormalizeLabel(t *testing.T) {
	blank := "   "
	if got, err := NormalizeLabel(&blank); err != nil || got != nil {
		t.Fatalf("blank label: %v %v", got, err)
	}
	good := " Hora de Costa Rica "
	if got, err := NormalizeLabel(&good); err != nil || *got != "Hora de Costa Rica" {
		t.Fatalf("good label: %v %v", got, err)
	}
	bad := "line1\nline2"
	if _, err := NormalizeLabel(&bad); err == nil {
		t.Fatal("a label with a new line must fail")
	}
}

// A company in New York while the server clock says another zone.
func TestCompanyNowAndTodayInNewYork(t *testing.T) {
	// 2026-11-05 03:30 UTC = 2026-11-04 22:30 in New York (GMT-5, winter)
	//                      = 2026-11-04 21:30 in Costa Rica.
	now := time.Date(2026, 11, 5, 3, 30, 0, 0, time.UTC)
	useFakeZone(t, "America/New_York", nil, now)

	if got := CompanyToday("c1"); got != "2026-11-04" {
		t.Fatalf("CompanyToday = %s, want 2026-11-04", got)
	}
	if got := CompanyNow("c1").Format("15:04"); got != "22:30" {
		t.Fatalf("CompanyNow = %s, want 22:30", got)
	}
}

// Summer time: the same zone has another offset in July.
func TestZoneTextFollowsSummerTime(t *testing.T) {
	ny := LocationFor("America/New_York")
	winter := ZoneTextOn(ny, "", "2026-01-15", "09:00")
	summer := ZoneTextOn(ny, "", "2026-07-15", "09:00")
	if winter != "New York, GMT−5" || summer != "New York, GMT−4" {
		t.Fatalf("winter=%q summer=%q", winter, summer)
	}
	cr := LocationFor("America/Costa_Rica")
	if got := ZoneTextOn(cr, "Hora de Costa Rica", "2026-07-15", "09:00"); got != "Hora de Costa Rica, GMT−6" {
		t.Fatalf("Costa Rica text = %q", got)
	}
	if got := ZoneTextOn(LocationFor("Asia/Kolkata"), "", "2026-07-15", "09:00"); got != "Kolkata, GMT+5:30" {
		t.Fatalf("Kolkata text = %q", got)
	}
}

func TestLocationForFallsBackToDefault(t *testing.T) {
	for _, name := range []string{"", "Local", "Nowhere/City"} {
		if got := LocationFor(name).String(); got != DefaultTimeZone {
			t.Fatalf("LocationFor(%q) = %s", name, got)
		}
	}
}

func TestCacheAndForget(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	useFakeZone(t, "Europe/Madrid", nil, now)
	if CompanyLocation("c2").String() != "Europe/Madrid" {
		t.Fatal("first read")
	}
	loadZoneFromDB = func(string) (string, *string, error) { return "America/New_York", nil, nil }
	if CompanyLocation("c2").String() != "Europe/Madrid" {
		t.Fatal("the cache must keep the zone for a few minutes")
	}
	Forget("c2")
	if CompanyLocation("c2").String() != "America/New_York" {
		t.Fatal("Forget must reload the zone")
	}
}

func TestWallTime(t *testing.T) {
	ny := LocationFor("America/New_York")
	got := WallTime(ny, "2026-07-10", "09:00:00", time.Time{})
	if got.UTC().Format("2006-01-02 15:04") != "2026-07-10 13:00" {
		t.Fatalf("9:00 New York in July = %s UTC, want 13:00", got.UTC().Format("2006-01-02 15:04"))
	}
}
