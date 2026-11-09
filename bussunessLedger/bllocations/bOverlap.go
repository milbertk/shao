package bllocations

import (
	"fmt"
	"strings"
	"time"

	"github.com/milbertk/shao/bussunessLedger/bltimezone"
	"github.com/milbertk/shao/dataBasesLedger/booking"
	"github.com/milbertk/shao/dataBasesLedger/locations"
)

// "One location at a time" (2026-11-07). When the company can NOT attend in
// more than one location at the same time (the default), two blocks of hours
// in DIFFERENT locations must not overlap, in any schedule of the company.
// Blocks with no location are not checked (as before). The rule applies only
// while the company uses locations (2026-11-08).
//
// A block where the customer chooses the location (AnyLocation, "*") can end
// in any place, so it clashes with every other block that has a location or
// is also "customer chooses".

const (
	KindRecurring = "RECURRING"
	KindManual    = "MANUAL"
	maxReported   = 10
	// AnyLocation marks a block where the customer chooses the location.
	AnyLocation = "*"
)

// TimeBlock is one block of hours with a location.
type TimeBlock struct {
	Kind         string // RECURRING | MANUAL
	ScheduleID   string
	ScheduleName string
	DayOfWeek    string // RECURRING: MONDAY..SUNDAY
	Date         string // MANUAL: YYYY-MM-DD
	AllDay       bool
	StartTime    string // HH:MM
	EndTime      string // HH:MM
	LocationID   string // AnyLocation = the customer chooses
	LocationName string
}

// LocationOverlap is one clash: the new block and the block it clashes with.
type LocationOverlap struct {
	DayOfWeek           string `json:"dayOfWeek,omitempty"`
	Date                string `json:"date,omitempty"`
	AllDay              bool   `json:"allDay,omitempty"`
	StartTime           string `json:"startTime,omitempty"`
	EndTime             string `json:"endTime,omitempty"`
	LocationName        string `json:"locationName"`
	CustomerChoice      bool   `json:"customerChoice,omitempty"` // the new block: customer chooses
	OtherScheduleName   string `json:"otherScheduleName"`
	OtherDayOfWeek      string `json:"otherDayOfWeek,omitempty"`
	OtherDate           string `json:"otherDate,omitempty"`
	OtherAllDay         bool   `json:"otherAllDay,omitempty"`
	OtherStartTime      string `json:"otherStartTime,omitempty"`
	OtherEndTime        string `json:"otherEndTime,omitempty"`
	OtherLocationName   string `json:"otherLocationName"`
	OtherCustomerChoice bool   `json:"otherCustomerChoice,omitempty"` // the other block: customer chooses
}

// LocationOverlapError is returned when hours clash in two locations.
// Handlers answer 409 with code LOCATION_OVERLAP and the list.
type LocationOverlapError struct {
	Overlaps []LocationOverlap
}

func (e *LocationOverlapError) Error() string {
	if e == nil || len(e.Overlaps) == 0 {
		return "location overlap"
	}
	message := "location overlap: " + e.Overlaps[0].Message()
	if more := len(e.Overlaps) - 1; more > 0 {
		message += fmt.Sprintf(" (and %d more)", more)
	}
	return message
}

// Message is the English sentence of one clash, e.g.
// "Monday 10:00–12:00 is already in Heredia (Consultation)".
func (o LocationOverlap) Message() string {
	when := blockWhen(o.OtherDayOfWeek, o.OtherDate, o.OtherAllDay, o.OtherStartTime, o.OtherEndTime)
	where := o.OtherLocationName
	if o.OtherCustomerChoice {
		where = "the location the customer chooses"
	}
	return fmt.Sprintf("%s is already in %s (%s)", when, where, o.OtherScheduleName)
}

var dayNames = map[string]string{
	"MONDAY": "Monday", "TUESDAY": "Tuesday", "WEDNESDAY": "Wednesday", "THURSDAY": "Thursday",
	"FRIDAY": "Friday", "SATURDAY": "Saturday", "SUNDAY": "Sunday",
}

func blockWhen(day, date string, allDay bool, start, end string) string {
	label := date
	if day != "" {
		label = dayNames[strings.ToUpper(day)]
		if label == "" {
			label = day
		}
	}
	if allDay || start == "" || end == "" {
		return label + " (all day)"
	}
	return label + " " + start + "–" + end
}

// minutes parses HH:MM; -1 when invalid.
func minutes(value string) int {
	t, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return -1
	}
	return t.Hour()*60 + t.Minute()
}

// span returns [start, end) in minutes; all-day = the whole day.
func (b TimeBlock) span() (int, int, bool) {
	if b.AllDay || (b.StartTime == "" && b.EndTime == "") {
		return 0, 24 * 60, true
	}
	start, end := minutes(b.StartTime), minutes(b.EndTime)
	if start < 0 || end < 0 || end <= start {
		return 0, 0, false
	}
	return start, end, true
}

// weekday of a MANUAL block, or its DayOfWeek for RECURRING.
func (b TimeBlock) weekday() string {
	if b.Kind == KindRecurring {
		return strings.ToUpper(strings.TrimSpace(b.DayOfWeek))
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(b.Date))
	if err != nil {
		return ""
	}
	return strings.ToUpper(d.Weekday().String())
}

// sameDay: weekly vs weekly by weekday, date vs date, weekly vs date by the
// weekday of the date.
func sameDay(a, b TimeBlock) bool {
	if a.Kind == KindManual && b.Kind == KindManual {
		return strings.TrimSpace(a.Date) != "" && strings.TrimSpace(a.Date) == strings.TrimSpace(b.Date)
	}
	wa, wb := a.weekday(), b.weekday()
	return wa != "" && wa == wb
}

// Clash tells if two blocks are in different locations at overlapping hours
// of the same day (pure, tested).
func Clash(a, b TimeBlock) bool {
	la, lb := strings.ToLower(strings.TrimSpace(a.LocationID)), strings.ToLower(strings.TrimSpace(b.LocationID))
	if la == "" || lb == "" {
		return false
	}
	if la == lb && la != AnyLocation {
		return false
	}
	if !sameDay(a, b) {
		return false
	}
	as, ae, okA := a.span()
	bs, be, okB := b.span()
	if !okA || !okB {
		return false
	}
	return as < be && bs < ae
}

func overlapOf(a, b TimeBlock) LocationOverlap {
	return LocationOverlap{
		DayOfWeek:           strings.ToUpper(a.DayOfWeek),
		Date:                a.Date,
		AllDay:              a.AllDay,
		StartTime:           a.StartTime,
		EndTime:             a.EndTime,
		LocationName:        a.LocationName,
		CustomerChoice:      a.LocationID == AnyLocation,
		OtherScheduleName:   b.ScheduleName,
		OtherDayOfWeek:      strings.ToUpper(b.DayOfWeek),
		OtherDate:           b.Date,
		OtherAllDay:         b.AllDay,
		OtherStartTime:      b.StartTime,
		OtherEndTime:        b.EndTime,
		OtherLocationName:   b.LocationName,
		OtherCustomerChoice: b.LocationID == AnyLocation,
	}
}

// FindLocationOverlaps returns the clashes of the blocks among themselves
// and against the others (pure, tested). At most 10 are returned.
func FindLocationOverlaps(blocks, others []TimeBlock) []LocationOverlap {
	found := make([]LocationOverlap, 0)
	for i := range blocks {
		for j := i + 1; j < len(blocks); j++ {
			if Clash(blocks[i], blocks[j]) {
				found = append(found, overlapOf(blocks[j], blocks[i]))
				if len(found) >= maxReported {
					return found
				}
			}
		}
		for _, other := range others {
			if Clash(blocks[i], other) {
				found = append(found, overlapOf(blocks[i], other))
				if len(found) >= maxReported {
					return found
				}
			}
		}
	}
	return found
}

// BlocksFromRows converts the stored hours into blocks.
func BlocksFromRows(rows []booking.LocationTimeRow) []TimeBlock {
	blocks := make([]TimeBlock, 0, len(rows))
	for _, r := range rows {
		blocks = append(blocks, TimeBlock{
			Kind:         r.Kind,
			ScheduleID:   r.BookingConfigID,
			ScheduleName: r.ScheduleName,
			DayOfWeek:    r.DayOfWeek,
			Date:         r.SlotDate,
			AllDay:       r.AllDay,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			LocationID:   r.LocationID,
			LocationName: r.LocationName,
		})
	}
	return blocks
}

// LocationsEnabled tells if the company uses locations ("We use locations",
// 2026-11-08, default false). OFF = AlephK works as before locations.
func LocationsEnabled(companyID string) (bool, error) {
	settings, err := locations.SelectBookingSettings(companyID)
	if err != nil {
		return false, err
	}
	return settings.LocationsEnabled, nil
}

// OneLocationAtATime tells if the "one location at a time" rule applies:
// the company uses locations AND can not attend in several at the same time.
func OneLocationAtATime(companyID string) (bool, error) {
	settings, err := locations.SelectBookingSettings(companyID)
	if err != nil {
		return false, err
	}
	return RuleActive(settings), nil
}

// RuleActive: the "one location at a time" rule for these settings (pure).
func RuleActive(settings locations.BookingSettingsS) bool {
	return settings.LocationsEnabled && !settings.LocationsSameTime
}

// CheckScheduleBlocks checks the blocks of the schedule being saved against
// the other schedules of the company. Nothing to check when the company does
// not use locations or can attend in several locations at the same time.
// Returns *LocationOverlapError.
func CheckScheduleBlocks(companyID, scheduleID string, blocks []TimeBlock) error {
	if len(blocks) == 0 {
		return nil
	}
	active, err := OneLocationAtATime(companyID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	rows, err := booking.SelectCompanyLocationTimes(companyID, scheduleID, bltimezone.CompanyToday(companyID), "")
	if err != nil {
		return err
	}
	if overlaps := FindLocationOverlaps(blocks, BlocksFromRows(rows)); len(overlaps) > 0 {
		return &LocationOverlapError{Overlaps: overlaps}
	}
	return nil
}

// checkEnableWithoutOverlap: enabling a disabled location must not create
// clashes when the company is in one location at a time.
func checkEnableWithoutOverlap(companyID, locationID string) error {
	active, err := OneLocationAtATime(companyID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	rows, err := booking.SelectCompanyLocationTimes(companyID, "", bltimezone.CompanyToday(companyID), locationID)
	if err != nil {
		return err
	}
	var mine, others []TimeBlock
	for _, b := range BlocksFromRows(rows) {
		if strings.EqualFold(b.LocationID, locationID) {
			mine = append(mine, b)
		} else {
			others = append(others, b)
		}
	}
	if overlaps := FindLocationOverlaps(mine, others); len(overlaps) > 0 {
		return &LocationOverlapError{Overlaps: overlaps}
	}
	return nil
}

// currentOverlaps returns the clashes that exist today among all schedules.
func currentOverlaps(companyID string) ([]LocationOverlap, error) {
	rows, err := booking.SelectCompanyLocationTimes(companyID, "", bltimezone.CompanyToday(companyID), "")
	if err != nil {
		return nil, err
	}
	return FindLocationOverlaps(BlocksFromRows(rows), nil), nil
}
