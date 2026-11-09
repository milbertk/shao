package blbooking

import (
	"fmt"
	"strings"
)

// Rules of the hours of one schedule (2026-11-08).
//
//   - One mode only: a schedule is either weekly (RECURRING) or by dates
//     (MANUAL). When saved, the hours of the other mode are deleted (dates
//     from today forward; past dates stay as history). Appointments are never
//     touched.
//   - Two ranges of the same schedule can overlap only in two DIFFERENT
//     locations (the database rule ex_*_no_overlap after
//     sql/migration_booking_overlap_by_location.sql). Checked here BEFORE
//     anything is deleted, so a refused save never loses the saved hours.
//     Whether different locations may overlap is decided by the company
//     switch "same time" (bllocations, checked before this).

// RangesOverlapError: two ranges of this schedule overlap in the same
// location (or both without a fixed location). Handlers answer 409.
type RangesOverlapError struct {
	When string // weekday (MONDAY) or date (2026-11-09)
	A, B string // "09:00-12:00"
}

func (e *RangesOverlapError) Error() string {
	if e == nil || e.When == "" {
		return "these hours overlap with another range of this schedule in the same location"
	}
	return fmt.Sprintf("these hours overlap with another range of this schedule in the same location: %s %s and %s",
		dayOrDate(e.When), e.A, e.B)
}

var weekdayNames = map[string]string{
	"MONDAY": "Monday", "TUESDAY": "Tuesday", "WEDNESDAY": "Wednesday", "THURSDAY": "Thursday",
	"FRIDAY": "Friday", "SATURDAY": "Saturday", "SUNDAY": "Sunday",
}

func dayOrDate(value string) string {
	if name, ok := weekdayNames[strings.ToUpper(value)]; ok {
		return name
	}
	return value
}

// KeepActiveMode drops the hours of the mode that is not used, so saving
// deletes them (pure, tested).
func KeepActiveMode(input *BookingBundle) {
	if strings.EqualFold(strings.TrimSpace(input.Config.AvailabilityMode), "MANUAL") {
		input.RecurringRanges = nil
	} else {
		input.ManualSlots = nil
	}
}

// DropPastDates leaves out the dates before today (pure, tested). Past dates
// are history: saving never deletes them, so sending them again would insert
// them twice (the database refuses it). 2026-11-08.
func DropPastDates(input *BookingBundle, today string) {
	today = strings.TrimSpace(today)
	if today == "" || len(input.ManualSlots) == 0 {
		return
	}
	kept := input.ManualSlots[:0]
	for _, s := range input.ManualSlots {
		if strings.TrimSpace(s.SlotDate) >= today {
			kept = append(kept, s)
		}
	}
	input.ManualSlots = kept
}

// hhmm keeps "HH:MM" of "HH:MM[:SS]".
func hhmm(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 5 {
		return value[:5]
	}
	return value
}

// placeKey: the location as the database rule sees it ("" = no fixed
// location, including "Customer chooses").
func placeKey(id *string) string {
	if id == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*id))
}

type hourBlock struct {
	group, place, start, end string
}

func firstOverlap(blocks []hourBlock) *RangesOverlapError {
	for i := range blocks {
		for j := i + 1; j < len(blocks); j++ {
			a, b := blocks[i], blocks[j]
			if a.group != b.group || a.place != b.place {
				continue
			}
			if a.start < b.end && b.start < a.end {
				return &RangesOverlapError{When: a.group, A: a.start + "-" + a.end, B: b.start + "-" + b.end}
			}
		}
	}
	return nil
}

// CheckSameLocationOverlaps refuses two ranges of the schedule that overlap
// in the same location (pure, tested). All-day dates are not compared.
func CheckSameLocationOverlaps(input BookingBundle) error {
	weekly := make([]hourBlock, 0, len(input.RecurringRanges))
	for _, r := range input.RecurringRanges {
		weekly = append(weekly, hourBlock{
			group: strings.ToUpper(strings.TrimSpace(r.DayOfWeek)), place: placeKey(r.LocationID),
			start: hhmm(r.StartTime), end: hhmm(r.EndTime),
		})
	}
	if e := firstOverlap(weekly); e != nil {
		return e
	}
	dates := make([]hourBlock, 0, len(input.ManualSlots))
	for _, s := range input.ManualSlots {
		if s.AllDay || s.StartTime == nil || s.EndTime == nil {
			continue
		}
		dates = append(dates, hourBlock{
			group: strings.TrimSpace(s.SlotDate), place: placeKey(s.LocationID),
			start: hhmm(*s.StartTime), end: hhmm(*s.EndTime),
		})
	}
	if e := firstOverlap(dates); e != nil {
		return e
	}
	return nil
}

// overlapOr turns a database exclusion error (code 23P01) into the friendly
// RangesOverlapError; other errors keep their text.
func overlapOr(err error, fallback string) error {
	if err == nil {
		return nil
	}
	m := strings.ToLower(err.Error())
	if strings.Contains(m, "exclusion constraint") || strings.Contains(m, "23p01") {
		return &RangesOverlapError{}
	}
	return fmt.Errorf("%s: %w", fallback, err)
}
