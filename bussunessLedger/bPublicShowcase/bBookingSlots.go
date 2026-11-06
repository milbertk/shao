package blpublicshowcase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// AvailableSlot is one bookable slot returned to the public frontend, with the
// remaining capacity already computed (capacity - taken).
type AvailableSlot struct {
	Source    string `json:"source"`    // RECURRING | MANUAL
	StartTime string `json:"startTime"` // "HH:MM" ("" for all-day)
	EndTime   string `json:"endTime"`   // "HH:MM" ("" for all-day)
	AllDay    bool   `json:"allDay"`
	Capacity  int    `json:"capacity"`  // total capacity of the slot
	Taken     int    `json:"taken"`     // appointments already taken
	Remaining int    `json:"remaining"` // capacity - taken (>= 0)
}

// DayAvailability is the response for one date.
type DayAvailability struct {
	Date      string          `json:"date"`
	Mode      string          `json:"mode"`      // RECURRING | MANUAL
	Available bool            `json:"available"` // false if disabled/holiday/out of window
	Reason    string          `json:"reason"`    // why not available (optional)
	Slots     []AvailableSlot `json:"slots"`     // only slots WITH remaining > 0
	// Time zone of the company (2026-11-04): the hours are in this zone.
	TimeZone string `json:"timeZone"`
}

const dateLayout = "2006-01-02"

// weekdayEnum maps Go's weekday to the DB enum MONDAY..SUNDAY.
func weekdayEnum(t time.Time) string {
	switch t.Weekday() {
	case time.Monday:
		return "MONDAY"
	case time.Tuesday:
		return "TUESDAY"
	case time.Wednesday:
		return "WEDNESDAY"
	case time.Thursday:
		return "THURSDAY"
	case time.Friday:
		return "FRIDAY"
	case time.Saturday:
		return "SATURDAY"
	default:
		return "SUNDAY"
	}
}

// bookingWindowDays converts the enum to a number of days ahead.
func bookingWindowDays(window string) int {
	switch strings.ToUpper(strings.TrimSpace(window)) {
	case "ONE_WEEK":
		return 7
	case "ONE_MONTH":
		return 31
	case "THREE_MONTHS":
		return 92
	case "SIX_MONTHS":
		return 183
	case "ONE_YEAR":
		return 366
	default:
		return 31
	}
}

// parseHM parses "HH:MM" into minutes since midnight. Returns -1 on error.
func parseHM(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return -1
	}
	t, err := time.Parse("15:04", value)
	if err != nil {
		return -1
	}
	return t.Hour()*60 + t.Minute()
}

// minutesToHM turns minutes-since-midnight into "HH:MM".
func minutesToHM(total int) string {
	h := total / 60
	m := total % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// splitRange divides [start,end) into consecutive sub-slots of `duration`
// minutes, discarding any leftover that does not fit a full duration.
// Each returned pair is (startHM, endHM). If duration <= 0, returns the whole
// range as a single sub-slot.
func splitRange(startHM, endHM string, duration int) [][2]string {
	start := parseHM(startHM)
	end := parseHM(endHM)
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	if duration <= 0 {
		return [][2]string{{startHM, endHM}}
	}

	out := make([][2]string, 0)
	for cursor := start; cursor+duration <= end; cursor += duration {
		out = append(out, [2]string{
			minutesToHM(cursor),
			minutesToHM(cursor + duration),
		})
	}
	return out
}

// BGetAvailabilityForDate computes the bookable slots for a company + product
// on a specific date. Public/read-only (companyID comes from the URL).
//
//   - Loads the product's booking config (must be active).
//   - Applies the booking window and the holiday disable range.
//   - RECURRING: uses the ranges for that weekday. MANUAL: uses that date.
//   - Subtracts appointments already taken; returns only slots with capacity.
func BGetAvailabilityForDate(companyID, productID, date string) (*DayAvailability, error) {
	companyID = strings.TrimSpace(companyID)
	productID = strings.TrimSpace(productID)
	date = strings.TrimSpace(date)
	if companyID == "" || productID == "" {
		return nil, errors.New("companyId and productId are required")
	}
	if date == "" {
		return nil, errors.New("date is required")
	}

	target, err := time.Parse(dateLayout, date)
	if err != nil {
		return nil, fmt.Errorf("invalid date (use YYYY-MM-DD): %w", err)
	}

	config, ok, err := booking.GetBookingConfigForProduct(companyID, productID)
	if err != nil {
		return nil, err
	}

	result := &DayAvailability{Date: date, Slots: []AvailableSlot{}}

	// No config, or inactive -> nothing available.
	if !ok || !config.Active {
		result.Available = false
		result.Reason = "no active booking configuration"
		return result, nil
	}
	result.Mode = config.AvailabilityMode

	// Booking window: today .. today + windowDays.
	// "Today" and "now" are the COMPANY's (its time zone, 2026-11-04), not the
	// server's: the server runs in UTC. Build "today" from the date components
	// in that zone (Truncate would use UTC and shift the day).
	nowLocal := bltimezone.CompanyNow(companyID)
	result.TimeZone = nowLocal.Location().String()
	today := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, nowLocal.Location())
	targetDay := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, nowLocal.Location())

	if targetDay.Before(today) {
		result.Available = false
		result.Reason = "date is in the past"
		return result, nil
	}
	maxDay := today.AddDate(0, 0, bookingWindowDays(config.BookingWindow))
	if targetDay.After(maxDay) {
		result.Available = false
		result.Reason = "date is beyond the booking window"
		return result, nil
	}

	// Holiday disable range.
	if config.DisableHolidaysEnabled && config.DisabledFrom != "" && config.DisabledTo != "" {
		from, e1 := time.Parse(dateLayout, config.DisabledFrom)
		to, e2 := time.Parse(dateLayout, config.DisabledTo)
		if e1 == nil && e2 == nil {
			if !targetDay.Before(from) && !targetDay.After(to) {
				result.Available = false
				result.Reason = "date is disabled (holiday)"
				return result, nil
			}
		}
	}

	// Load the raw slots depending on the mode.
	var rawSlots []booking.AvailableSlotRow
	if strings.ToUpper(config.AvailabilityMode) == "MANUAL" {
		rawSlots, err = booking.GetManualSlotsForDate(
			companyID, config.BookingConfigID, date, config.GeneralAppointments,
		)
	} else {
		weekday := weekdayEnum(target)
		rawSlots, err = booking.GetRecurringSlotsForWeekday(
			companyID, config.BookingConfigID, weekday, config.GeneralAppointments,
		)
	}
	if err != nil {
		return nil, err
	}

	// For each raw slot: if it is all-day, offer it as one slot; otherwise
	// SPLIT the range into sub-slots of `appointment_duration_minutes` (leftover
	// discarded), and for EACH sub-slot subtract the appointments that overlap
	// exactly that sub-slot. Only sub-slots with remaining capacity are kept.
	duration := config.AppointmentDurationMinutes

	// When the target date is TODAY, hide sub-slots whose START time already
	// passed. For future dates, show everything.
	isToday := targetDay.Equal(today)
	nowMinutes := nowLocal.Hour()*60 + nowLocal.Minute()

	for _, raw := range rawSlots {
		// All-day slot: keep as a single entry, count the whole day.
		if raw.AllDay || raw.StartTime == "" || raw.EndTime == "" {
			taken, err := booking.CountTakenAppointments(
				companyID, productID, date, "00:00", "23:59",
			)
			if err != nil {
				return nil, err
			}
			remaining := raw.Capacity - taken
			if remaining < 0 {
				remaining = 0
			}
			if remaining <= 0 {
				continue
			}
			result.Slots = append(result.Slots, AvailableSlot{
				Source:    raw.Source,
				StartTime: "",
				EndTime:   "",
				AllDay:    true,
				Capacity:  raw.Capacity,
				Taken:     taken,
				Remaining: remaining,
			})
			continue
		}

		// Timed range: split into sub-slots of `duration` minutes.
		subSlots := splitRange(raw.StartTime, raw.EndTime, duration)
		for _, sub := range subSlots {
			subStart := sub[0]
			subEnd := sub[1]

			// Today: skip sub-slots whose start time already passed.
			if isToday && parseHM(subStart) <= nowMinutes {
				continue
			}

			taken, err := booking.CountTakenAppointments(
				companyID, productID, date, subStart, subEnd,
			)
			if err != nil {
				return nil, err
			}

			remaining := raw.Capacity - taken
			if remaining < 0 {
				remaining = 0
			}
			if remaining <= 0 {
				continue // this sub-slot is full
			}

			result.Slots = append(result.Slots, AvailableSlot{
				Source:    raw.Source,
				StartTime: subStart,
				EndTime:   subEnd,
				AllDay:    false,
				Capacity:  raw.Capacity,
				Taken:     taken,
				Remaining: remaining,
			})
		}
	}

	result.Available = len(result.Slots) > 0
	if !result.Available && result.Reason == "" {
		result.Reason = "no slots available for this date"
	}
	return result, nil
}
