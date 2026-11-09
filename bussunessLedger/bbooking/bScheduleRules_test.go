package blbooking

import (
	"errors"
	"strings"
	"testing"

	"github.com/milbertk/shao/dataBasesLedger/booking"
)

func sp(v string) *string { return &v }

func TestKeepActiveMode(t *testing.T) {
	b := BookingBundle{RecurringRanges: []booking.BookingRecurringRangeS{{}}, ManualSlots: []booking.BookingManualSlotS{{}}}
	b.Config.AvailabilityMode = "MANUAL"
	KeepActiveMode(&b)
	if b.RecurringRanges != nil || len(b.ManualSlots) != 1 {
		t.Fatalf("manual: %+v", b)
	}
	b = BookingBundle{RecurringRanges: []booking.BookingRecurringRangeS{{}}, ManualSlots: []booking.BookingManualSlotS{{}}}
	b.Config.AvailabilityMode = "RECURRING"
	KeepActiveMode(&b)
	if b.ManualSlots != nil || len(b.RecurringRanges) != 1 {
		t.Fatalf("weekly: %+v", b)
	}
}

func TestCheckSameLocationOverlaps(t *testing.T) {
	her, sj := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	weekly := func(day, start, end string, loc *string) booking.BookingRecurringRangeS {
		return booking.BookingRecurringRangeS{DayOfWeek: day, StartTime: start, EndTime: end, LocationID: loc}
	}
	date := func(d, start, end string, loc *string) booking.BookingManualSlotS {
		return booking.BookingManualSlotS{SlotDate: d, StartTime: sp(start), EndTime: sp(end), LocationID: loc}
	}
	ok := []BookingBundle{
		{RecurringRanges: []booking.BookingRecurringRangeS{weekly("MONDAY", "09:00", "12:00", &her), weekly("MONDAY", "09:00", "12:00", &sj)}},
		{RecurringRanges: []booking.BookingRecurringRangeS{weekly("MONDAY", "09:00", "12:00", nil), weekly("MONDAY", "12:00", "13:00", nil)}},
		{RecurringRanges: []booking.BookingRecurringRangeS{weekly("MONDAY", "09:00", "12:00", nil), weekly("TUESDAY", "09:00", "12:00", nil)}},
		{ManualSlots: []booking.BookingManualSlotS{date("2026-11-09", "09:00:00", "12:00:00", &her), date("2026-11-09", "10:00", "11:00", &sj)}},
		{ManualSlots: []booking.BookingManualSlotS{{SlotDate: "2026-11-09", AllDay: true}, date("2026-11-09", "10:00", "11:00", nil)}},
	}
	for i, b := range ok {
		if err := CheckSameLocationOverlaps(b); err != nil {
			t.Fatalf("ok %d: %v", i, err)
		}
	}
	bad := []BookingBundle{
		{RecurringRanges: []booking.BookingRecurringRangeS{weekly("MONDAY", "09:00", "12:00", &her), weekly("MONDAY", "11:00", "13:00", sp(strings.ToUpper(her)))}},
		{RecurringRanges: []booking.BookingRecurringRangeS{weekly("MONDAY", "09:00", "12:00", nil), weekly("MONDAY", "11:00", "13:00", nil)}},
		{ManualSlots: []booking.BookingManualSlotS{date("2026-11-09", "09:00", "12:00", &sj), date("2026-11-09", "10:00", "11:00", &sj)}},
	}
	for i, b := range bad {
		var e *RangesOverlapError
		if err := CheckSameLocationOverlaps(b); !errors.As(err, &e) {
			t.Fatalf("bad %d: %v", i, err)
		}
	}
	err := CheckSameLocationOverlaps(bad[0])
	if !strings.Contains(err.Error(), "Monday 09:00-12:00 and 11:00-13:00") || !strings.Contains(strings.ToLower(err.Error()), "overlap") {
		t.Fatal(err)
	}
	if e := overlapOr(errors.New(`pq: conflicting key value violates exclusion constraint "ex_manual_no_overlap"`), "x"); !strings.Contains(e.Error(), "same location") {
		t.Fatal(e)
	}
	if e := overlapOr(errors.New("boom"), "failed to insert"); e.Error() != "failed to insert: boom" {
		t.Fatal(e)
	}
}

func TestDropPastDates(t *testing.T) {
	b := BookingBundle{ManualSlots: []booking.BookingManualSlotS{
		{SlotDate: "2026-11-01"}, {SlotDate: "2026-11-08"}, {SlotDate: "2026-11-09"},
	}}
	DropPastDates(&b, "2026-11-08")
	if len(b.ManualSlots) != 2 || b.ManualSlots[0].SlotDate != "2026-11-08" || b.ManualSlots[1].SlotDate != "2026-11-09" {
		t.Fatalf("%+v", b.ManualSlots)
	}
	DropPastDates(&b, "")
	if len(b.ManualSlots) != 2 {
		t.Fatal("no today: keep all")
	}
}
