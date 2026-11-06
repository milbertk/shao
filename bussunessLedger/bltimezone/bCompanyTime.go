// Package bltimezone gives every company its own clock (2026-11-04).
//
// The server and the database work in UTC. Each company chooses its time zone
// in "Company page" (tbl_page_configuration.time_zone, an IANA name such as
// America/Costa_Rica or America/New_York). Saved appointment and slot hours
// are wall-clock hours of the company (14:00 stays 14:00); only the
// calculations ("now", "today", "until 2 hours before") use the zone.
// IANA names handle summer time automatically (New York is GMT-5 in winter
// and GMT-4 in summer); Costa Rica is GMT-6 all year.
package bltimezone

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
)

// DefaultTimeZone is used when a company has not chosen one (GMT-6).
const DefaultTimeZone = pageconfiguration.DefaultTimeZone

// MaxLabelLength is the maximum length of the optional zone text.
const MaxLabelLength = 60

// cacheTTL keeps a company's zone in memory for a short time, so busy pages
// (store slots, emails) do not read it from the database every time.
const cacheTTL = 5 * time.Minute

// Package hooks, replaced in tests.
var (
	nowFunc        = time.Now
	loadZoneFromDB = pageconfiguration.SelectTimeZone
)

type cachedZone struct {
	location *time.Location
	label    string
	until    time.Time
}

var (
	cacheMu sync.Mutex
	cache   = map[string]cachedZone{}
)

// ErrInvalidTimeZone is returned for a name that is not a known IANA zone.
var ErrInvalidTimeZone = errors.New("invalid timeZone: choose one zone from the list")

// NormalizeTimeZone trims the name, uses DefaultTimeZone when it is empty and
// checks that it is a real zone. "Local" is refused: it would mean the
// server's zone.
func NormalizeTimeZone(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return DefaultTimeZone, nil
	}
	if len(name) > 64 || strings.EqualFold(name, "Local") {
		return "", ErrInvalidTimeZone
	}
	if _, err := time.LoadLocation(name); err != nil {
		return "", ErrInvalidTimeZone
	}
	return name, nil
}

// NormalizeLabel trims the optional zone text (max 60 characters, no control
// characters). Empty = nil.
func NormalizeLabel(label *string) (*string, error) {
	if label == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*label)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > MaxLabelLength {
		return nil, fmt.Errorf("invalid timeZoneLabel: maximum length is %d characters", MaxLabelLength)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return nil, errors.New("invalid timeZoneLabel: use one line of text")
		}
	}
	return &value, nil
}

// LocationFor returns the zone with that name, or the default zone when the
// name is empty or unknown. It never returns nil.
func LocationFor(name string) *time.Location {
	if loc, err := time.LoadLocation(strings.TrimSpace(name)); err == nil && strings.TrimSpace(name) != "" && !strings.EqualFold(strings.TrimSpace(name), "Local") {
		return loc
	}
	return DefaultLocation()
}

// DefaultLocation is Costa Rica (GMT-6). Falls back to a fixed GMT-6 zone if
// the time zone database is missing (main.go embeds it with time/tzdata).
func DefaultLocation() *time.Location {
	if loc, err := time.LoadLocation(DefaultTimeZone); err == nil {
		return loc
	}
	return time.FixedZone("GMT-6", -6*60*60)
}

// CompanyZone returns the company's zone and its optional label. Errors are
// not fatal: the default zone is used and the next call tries again.
func CompanyZone(companyID string) (*time.Location, string) {
	companyID = strings.TrimSpace(companyID)
	now := nowFunc()

	cacheMu.Lock()
	if entry, ok := cache[companyID]; ok && now.Before(entry.until) {
		cacheMu.Unlock()
		return entry.location, entry.label
	}
	cacheMu.Unlock()

	if companyID == "" {
		return DefaultLocation(), ""
	}

	name, label, err := loadZoneFromDB(companyID)
	location := LocationFor(name)
	text := ""
	if label != nil {
		text = strings.TrimSpace(*label)
	}
	if err != nil {
		return location, text // do not cache a database error
	}

	cacheMu.Lock()
	cache[companyID] = cachedZone{location: location, label: text, until: now.Add(cacheTTL)}
	cacheMu.Unlock()
	return location, text
}

// Now is the current moment (replaceable in tests). Compare it with times
// built in a company zone; time.Time compares instants, not wall clocks.
func Now() time.Time {
	return nowFunc()
}

// OverrideForTests replaces the clock and the zone lookup (tests of other
// packages: "a company in New York"). Call the returned function to restore.
func OverrideForTests(load func(companyID string) (string, *string, error), now func() time.Time) (restore func()) {
	cacheMu.Lock()
	oldLoad, oldNow := loadZoneFromDB, nowFunc
	if load != nil {
		loadZoneFromDB = load
	}
	if now != nil {
		nowFunc = now
	}
	cache = map[string]cachedZone{}
	cacheMu.Unlock()
	return func() {
		cacheMu.Lock()
		loadZoneFromDB, nowFunc = oldLoad, oldNow
		cache = map[string]cachedZone{}
		cacheMu.Unlock()
	}
}

// CompanyLocation returns the company's zone.
func CompanyLocation(companyID string) *time.Location {
	location, _ := CompanyZone(companyID)
	return location
}

// CompanyNow is the current time on the company's clock.
func CompanyNow(companyID string) time.Time {
	return nowFunc().In(CompanyLocation(companyID))
}

// CompanyToday is the company's current date (YYYY-MM-DD).
func CompanyToday(companyID string) string {
	return CompanyNow(companyID).Format("2006-01-02")
}

// Forget removes a company from the cache (called after "Company page" is
// saved, so a new zone is used at once).
func Forget(companyID string) {
	cacheMu.Lock()
	delete(cache, strings.TrimSpace(companyID))
	cacheMu.Unlock()
}

// OffsetText renders the GMT offset of a zone at a moment: "GMT-6",
// "GMT+5:30", "GMT" (the minus is a real minus sign for emails: "GMT−6").
func OffsetText(at time.Time) string {
	_, seconds := at.Zone()
	if seconds == 0 {
		return "GMT"
	}
	sign := "+"
	if seconds < 0 {
		sign = "−"
		seconds = -seconds
	}
	hours, minutes := seconds/3600, (seconds%3600)/60
	if minutes == 0 {
		return fmt.Sprintf("GMT%s%d", sign, hours)
	}
	return fmt.Sprintf("GMT%s%d:%02d", sign, hours, minutes)
}

// CityName turns an IANA name into a short place name:
// America/Costa_Rica -> "Costa Rica", America/Argentina/Buenos_Aires ->
// "Buenos Aires", UTC -> "UTC".
func CityName(location *time.Location) string {
	name := location.String()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return strings.ReplaceAll(name, "_", " ")
}

// ZoneText is what customers read next to an hour: the company label (or the
// city) and the GMT offset ON THAT DATE, e.g. "Hora de Costa Rica, GMT−6".
func ZoneText(location *time.Location, label string, at time.Time) string {
	place := strings.TrimSpace(label)
	if place == "" {
		place = CityName(location)
	}
	return place + ", " + OffsetText(at.In(location))
}

// ZoneTextOn is ZoneText for a wall-clock date and hour of the company
// ("2026-07-10", "09:00"); a bad date uses the current moment.
func ZoneTextOn(location *time.Location, label, date, hour string) string {
	return ZoneText(location, label, WallTime(location, date, hour, nowFunc()))
}

// WallTime builds the moment of a company date + hour ("2026-07-10" +
// "09:00") in its zone. An empty hour means the start of the day. On error
// it returns fallback.
func WallTime(location *time.Location, date, hour string, fallback time.Time) time.Time {
	hour = strings.TrimSpace(hour)
	if len(hour) > 5 {
		hour = hour[:5]
	}
	if hour == "" {
		hour = "00:00"
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(date)+" "+hour, location)
	if err != nil {
		return fallback
	}
	return t
}
