package bllocations

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/locations"
)

// BookingSettingsInput is what the Locations screen sends: the company
// switches "We use locations" (2026-11-08) and "We can attend in more than
// one location at the same time". A switch not sent keeps its value.
type BookingSettingsInput struct {
	LocationsEnabled  *bool `json:"locationsEnabled"`
	LocationsSameTime *bool `json:"locationsSameTime"`
}

// MergeSettings applies the input to the current settings and tells if the
// "one location at a time" rule becomes active with this change (pure, tested).
func MergeSettings(current locations.BookingSettingsS, input BookingSettingsInput) (locations.BookingSettingsS, bool) {
	next := current
	if input.LocationsEnabled != nil {
		next.LocationsEnabled = *input.LocationsEnabled
	}
	if input.LocationsSameTime != nil {
		next.LocationsSameTime = *input.LocationsSameTime
	}
	return next, RuleActive(next) && !RuleActive(current)
}

// BGetBookingSettings returns the company booking settings (defaults when
// none were saved).
func BGetBookingSettings(companyID string) (*locations.BookingSettingsS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	settings, err := locations.SelectBookingSettings(companyID)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// BSaveBookingSettings saves the switches. When the change makes the "one
// location at a time" rule active (locations turned ON while "same time" is
// off, or "same time" turned OFF), it is refused while two schedules already
// have the same hours in two locations: the answer lists them
// (*LocationOverlapError) so they can be changed first.
func BSaveBookingSettings(companyID string, input BookingSettingsInput) (*locations.BookingSettingsS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if input.LocationsEnabled == nil && input.LocationsSameTime == nil {
		return nil, errors.New("locationsEnabled or locationsSameTime is required")
	}
	current, err := locations.SelectBookingSettings(companyID)
	if err != nil {
		return nil, err
	}
	next, ruleStarts := MergeSettings(current, input)
	if ruleStarts {
		overlaps, err := currentOverlaps(companyID)
		if err != nil {
			return nil, err
		}
		if len(overlaps) > 0 {
			return nil, &LocationOverlapError{Overlaps: overlaps}
		}
	}
	settings, err := locations.SaveBookingSettings(companyID, next.LocationsEnabled, next.LocationsSameTime)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// MapsURL returns an "Open in Maps" link (Google Maps, works on phones and
// computers, no key needed): the map point when there is one, otherwise the
// address. "" when the location has neither.
func MapsURL(address *string, latitude, longitude *float64) string {
	if latitude != nil && longitude != nil {
		return "https://www.google.com/maps/search/?api=1&query=" +
			strconv.FormatFloat(*latitude, 'f', 6, 64) + "," + strconv.FormatFloat(*longitude, 'f', 6, 64)
	}
	if address != nil && strings.TrimSpace(*address) != "" {
		return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(strings.TrimSpace(*address))
	}
	return ""
}
