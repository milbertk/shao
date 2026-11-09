package bllocations

import (
	"math"
	"strings"
	"testing"

	"github.com/milbertk/shao/dataBasesLedger/locations"
)

func ptr[T any](v T) *T { return &v }

const (
	heredia  = "11111111-1111-1111-1111-111111111111"
	sanPedro = "22222222-2222-2222-2222-222222222222"
)

func TestValidateLocationInput(t *testing.T) {
	in := LocationInput{Name: "  Oficina   Heredia ", Address: ptr("  "), Details: ptr(" Piso 2 ")}
	if err := ValidateLocationInput(&in); err != nil {
		t.Fatal(err)
	}
	if in.Name != "Oficina Heredia" || in.Address != nil || in.Details == nil || *in.Details != "Piso 2" {
		t.Fatalf("normalized: %+v", in)
	}

	bad := []LocationInput{
		{Name: ""},
		{Name: strings.Repeat("a", 121)},
		{Name: "X", Address: ptr(strings.Repeat("a", 301))},
		{Name: "X", Details: ptr(strings.Repeat("a", 501))},
		{Name: "X", Latitude: ptr(10.0)},                            // only one
		{Name: "X", Latitude: ptr(91.0), Longitude: ptr(1.0)},       // out of range
		{Name: "X", Latitude: ptr(1.0), Longitude: ptr(-181.0)},     // out of range
		{Name: "X", Latitude: ptr(0.0), Longitude: ptr(0.0)},        // not a place
		{Name: "X", Latitude: ptr(math.NaN()), Longitude: ptr(1.0)}, // NaN
		{Name: "X", LocationID: "not-a-uuid"},                       // bad id
	}
	for i, b := range bad {
		if err := ValidateLocationInput(&b); err == nil {
			t.Fatalf("case %d must fail: %+v", i, b)
		}
	}
	ok := LocationInput{Name: "NY", Latitude: ptr(40.7128), Longitude: ptr(-74.006), LocationID: heredia}
	if err := ValidateLocationInput(&ok); err != nil {
		t.Fatal(err)
	}
}

func TestMapsURL(t *testing.T) {
	if got := MapsURL(ptr("Av. 1, Heredia"), ptr(9.998), ptr(-84.117)); got != "https://www.google.com/maps/search/?api=1&query=9.998000,-84.117000" {
		t.Fatal(got)
	}
	if got := MapsURL(ptr("100 m norte & sur, Heredia"), nil, nil); got != "https://www.google.com/maps/search/?api=1&query=100+m+norte+%26+sur%2C+Heredia" {
		t.Fatal(got)
	}
	if MapsURL(nil, nil, nil) != "" || MapsURL(ptr(" "), nil, nil) != "" {
		t.Fatal("empty location must have no link")
	}
}

func block(kind, day, date, start, end, location, name string) TimeBlock {
	return TimeBlock{Kind: kind, DayOfWeek: day, Date: date, StartTime: start, EndTime: end,
		LocationID: location, LocationName: name, ScheduleName: "Consultation"}
}

func TestClash(t *testing.T) {
	monHer := block(KindRecurring, "MONDAY", "", "10:00", "12:00", heredia, "Heredia")
	cases := []struct {
		name string
		b    TimeBlock
		want bool
	}{
		{"same hours other location", block(KindRecurring, "MONDAY", "", "11:00", "13:00", sanPedro, "San Pedro"), true},
		{"same location", block(KindRecurring, "MONDAY", "", "10:00", "12:00", heredia, "Heredia"), false},
		{"touching hours", block(KindRecurring, "MONDAY", "", "12:00", "14:00", sanPedro, "San Pedro"), false},
		{"other day", block(KindRecurring, "TUESDAY", "", "10:00", "12:00", sanPedro, "San Pedro"), false},
		{"no location", block(KindRecurring, "MONDAY", "", "10:00", "12:00", "", ""), false},
		// 2026-11-09 is a Monday.
		{"manual monday vs weekly monday", block(KindManual, "", "2026-11-09", "09:00", "10:30", sanPedro, "San Pedro"), true},
		{"manual tuesday", block(KindManual, "", "2026-11-10", "09:00", "10:30", sanPedro, "San Pedro"), false},
		{"manual all day", TimeBlock{Kind: KindManual, Date: "2026-11-16", AllDay: true, LocationID: sanPedro}, true},
		{"upper/lower case id", block(KindRecurring, "MONDAY", "", "10:00", "12:00", strings.ToUpper(heredia), "Heredia"), false},
	}
	for _, c := range cases {
		if got := Clash(monHer, c.b); got != c.want {
			t.Fatalf("%s: got %v", c.name, got)
		}
		if got := Clash(c.b, monHer); got != c.want {
			t.Fatalf("%s (reversed): got %v", c.name, got)
		}
	}
	// manual vs manual: same date only.
	a := block(KindManual, "", "2026-11-09", "10:00", "11:00", heredia, "Heredia")
	if !Clash(a, block(KindManual, "", "2026-11-09", "10:30", "11:30", sanPedro, "")) ||
		Clash(a, block(KindManual, "", "2026-11-16", "10:30", "11:30", sanPedro, "")) {
		t.Fatal("manual vs manual")
	}
}

func TestFindLocationOverlapsAndMessage(t *testing.T) {
	existing := []TimeBlock{block(KindRecurring, "MONDAY", "", "10:00", "12:00", heredia, "Heredia")}
	newBlocks := []TimeBlock{
		block(KindRecurring, "MONDAY", "", "11:00", "12:00", sanPedro, "San Pedro"),
		block(KindRecurring, "FRIDAY", "", "08:00", "09:00", sanPedro, "San Pedro"),
		block(KindRecurring, "FRIDAY", "", "08:30", "10:00", heredia, "Heredia"), // clashes with the one above
	}
	found := FindLocationOverlaps(newBlocks, existing)
	if len(found) != 2 {
		t.Fatalf("want 2 clashes, got %d: %+v", len(found), found)
	}
	if msg := found[0].Message(); msg != "Monday 10:00–12:00 is already in Heredia (Consultation)" {
		t.Fatal(msg)
	}
	err := &LocationOverlapError{Overlaps: found}
	if !strings.Contains(err.Error(), "location overlap: ") || !strings.Contains(err.Error(), "(and 1 more)") {
		t.Fatal(err.Error())
	}
	if len(FindLocationOverlaps(nil, existing)) != 0 {
		t.Fatal("nothing new, nothing to report")
	}
	// At most 10 are reported.
	many := make([]TimeBlock, 0)
	for i := 0; i < 30; i++ {
		many = append(many, block(KindRecurring, "MONDAY", "", "10:00", "11:00", sanPedro, "San Pedro"))
	}
	if n := len(FindLocationOverlaps(many, existing)); n != maxReported {
		t.Fatalf("limit: %d", n)
	}
	allDay := LocationOverlap{OtherDate: "2026-11-20", OtherAllDay: true, OtherLocationName: "NY", OtherScheduleName: "Survey"}
	if allDay.Message() != "2026-11-20 (all day) is already in NY (Survey)" {
		t.Fatal(allDay.Message())
	}
}

func TestValidID(t *testing.T) {
	if !ValidID(heredia) || !ValidID(" "+strings.ToUpper(sanPedro)+" ") || ValidID("") || ValidID("1; DROP TABLE x") {
		t.Fatal("ValidID")
	}
}

func TestClashCustomerChooses(t *testing.T) {
	choice := block(KindRecurring, "MONDAY", "", "10:00", "12:00", AnyLocation, "")
	cases := []struct {
		name string
		b    TimeBlock
		want bool
	}{
		{"fixed location, even the same place", block(KindRecurring, "MONDAY", "", "11:00", "12:00", heredia, "Heredia"), true},
		{"another customer-chooses block", block(KindRecurring, "MONDAY", "", "11:00", "13:00", AnyLocation, ""), true},
		{"no location", block(KindRecurring, "MONDAY", "", "10:00", "12:00", "", ""), false},
		{"other hours", block(KindRecurring, "MONDAY", "", "12:00", "13:00", heredia, "Heredia"), false},
		{"date on that weekday", block(KindManual, "", "2026-11-09", "09:00", "10:30", sanPedro, "San Pedro"), true},
	}
	for _, c := range cases {
		if got := Clash(choice, c.b); got != c.want {
			t.Fatalf("%s: got %v", c.name, got)
		}
		if got := Clash(c.b, choice); got != c.want {
			t.Fatalf("%s (reversed): got %v", c.name, got)
		}
	}
	found := FindLocationOverlaps([]TimeBlock{block(KindRecurring, "MONDAY", "", "11:00", "12:00", heredia, "Heredia")}, []TimeBlock{choice})
	if len(found) != 1 || !found[0].OtherCustomerChoice ||
		found[0].Message() != "Monday 10:00–12:00 is already in the location the customer chooses (Consultation)" {
		t.Fatalf("%+v %s", found, found[0].Message())
	}
}

func TestMergeSettings(t *testing.T) {
	on, off := true, false
	type S = locations.BookingSettingsS
	cases := []struct {
		name       string
		current    S
		in         BookingSettingsInput
		want       S
		ruleStarts bool
	}{
		{"turn locations on (same time off)", S{}, BookingSettingsInput{LocationsEnabled: &on}, S{LocationsEnabled: true}, true},
		{"turn locations on with same time on", S{LocationsSameTime: true}, BookingSettingsInput{LocationsEnabled: &on}, S{LocationsEnabled: true, LocationsSameTime: true}, false},
		{"same time off while locations on", S{LocationsEnabled: true, LocationsSameTime: true}, BookingSettingsInput{LocationsSameTime: &off}, S{LocationsEnabled: true}, true},
		{"same time off while locations off", S{LocationsSameTime: true}, BookingSettingsInput{LocationsSameTime: &off}, S{}, false},
		{"locations off", S{LocationsEnabled: true}, BookingSettingsInput{LocationsEnabled: &off}, S{}, false},
		{"nothing changes", S{LocationsEnabled: true}, BookingSettingsInput{LocationsEnabled: &on}, S{LocationsEnabled: true}, false},
	}
	for _, c := range cases {
		got, starts := MergeSettings(c.current, c.in)
		if got.LocationsEnabled != c.want.LocationsEnabled || got.LocationsSameTime != c.want.LocationsSameTime || starts != c.ruleStarts {
			t.Fatalf("%s: got %+v %v", c.name, got, starts)
		}
	}
	if !RuleActive(S{LocationsEnabled: true}) || RuleActive(S{}) || RuleActive(S{LocationsEnabled: true, LocationsSameTime: true}) {
		t.Fatal("RuleActive")
	}
}
