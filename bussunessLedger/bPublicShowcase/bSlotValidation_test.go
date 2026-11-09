package blpublicshowcase

import (
	"sync"
	"testing"
	"time"

	"github.com/milbertk/shao/dataBasesLedger/booking"
	"github.com/milbertk/shao/dataBasesLedger/locations"
)

func TestHHMM(t *testing.T) {
	for in, want := range map[string]string{"9:00": "09:00", "09:30:00": "09:30", "": "", "14:05": "14:05"} {
		if got := hhmm(in); got != want {
			t.Fatalf("%q -> %q", in, got)
		}
	}
}

func TestTokens(t *testing.T) {
	if !validToken("a_0123456789abcdef0123456789abcdef0123456789abcdef") || validToken("x_0123456789abcdef0123") || validToken("a_../../etc") {
		t.Fatal("token validation")
	}
}

func TestLockSerializes(t *testing.T) {
	var wg sync.WaitGroup
	inside, max := 0, 0
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := lockSlot("c", "s", "2026-10-30")
			mu.Lock()
			inside++
			if inside > max {
				max = inside
			}
			mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if max != 1 || len(slotLocks) != 0 {
		t.Fatal("lock", max, len(slotLocks))
	}
}

func TestMatchSlotWithLocations(t *testing.T) {
	her, sp := "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	slots := []AvailableSlot{
		{StartTime: "10:00", EndTime: "11:00", Remaining: 1, LocationID: her},
		{StartTime: "10:00", EndTime: "11:00", Remaining: 2, LocationID: sp},
		{StartTime: "11:00", EndTime: "12:00", Remaining: 0, LocationID: sp},
		{StartTime: "12:00", EndTime: "13:00", Remaining: 1},
	}
	// The chosen location.
	if s, e, l, err := matchSlot(slots, "10:00", "11:00", sp); err != nil || s != "10:00" || e != "11:00" || l != sp {
		t.Fatalf("location chosen: %s %s %s %v", s, e, l, err)
	}
	// Upper case id from an old page still matches.
	if _, _, l, err := matchSlot(slots, "10:00", "11:00", "22222222-2222-2222-2222-222222222222"); err != nil || l != sp {
		t.Fatal("same id", err)
	}
	// No location sent (older store pages): the first free slot at that time.
	if _, _, l, err := matchSlot(slots, "10:00", "11:00", ""); err != nil || l != her {
		t.Fatalf("no location: %s %v", l, err)
	}
	// Full slot, or a time in another location -> taken.
	if _, _, _, err := matchSlot(slots, "11:00", "12:00", sp); err != ErrSlotTaken {
		t.Fatal("full slot must be taken")
	}
	if _, _, _, err := matchSlot(slots, "12:00", "13:00", her); err != ErrSlotTaken {
		t.Fatal("hours without that location must be refused")
	}
	// Slot without location.
	if _, _, l, err := matchSlot(slots, "12:00", "13:00", ""); err != nil || l != "" {
		t.Fatal("slot without location", err)
	}
	// All-day slot with a location.
	allDay := []AvailableSlot{{AllDay: true, Remaining: 1, LocationID: her}}
	if s, e, l, err := matchSlot(allDay, "", "", her); err != nil || s != "00:00" || e != "23:59" || l != her {
		t.Fatal("all day", s, e, l, err)
	}
}

func TestMatchSlotLocationsOff(t *testing.T) {
	// The company stopped using locations: the slots carry none and a
	// location sent by an open store page is ignored.
	slots := []AvailableSlot{{StartTime: "10:00", EndTime: "11:00", Remaining: 1}}
	if _, _, l, err := matchSlot(slots, "10:00", "11:00", "11111111-1111-1111-1111-111111111111"); err != nil || l != "" {
		t.Fatalf("locations off: %q %v", l, err)
	}
}

func TestExpandSlotCustomerChooses(t *testing.T) {
	her := locations.CompanyLocationS{LocationID: "11111111-1111-1111-1111-111111111111", Name: "Heredia", Active: true}
	sp := locations.CompanyLocationS{LocationID: "22222222-2222-2222-2222-222222222222", Name: "San Pedro", Active: true}
	off := locations.CompanyLocationS{LocationID: "33333333-3333-3333-3333-333333333333", Name: "Vieja", Active: false}
	byID := map[string]locations.CompanyLocationS{her.LocationID: her, sp.LocationID: sp, off.LocationID: off}
	active := activeLocations(byID)
	if len(active) != 2 || active[0].Name != "Heredia" || active[1].Name != "San Pedro" {
		t.Fatalf("active: %+v", active)
	}
	slot := AvailableSlot{StartTime: "10:00", EndTime: "11:00", Capacity: 1, Remaining: 1}
	copies := expandSlot(slot, booking.AvailableSlotRow{CustomerChoice: true}, byID, active)
	if len(copies) != 2 || !copies[0].CustomerChoosesLocation || copies[0].LocationName != "Heredia" || copies[1].LocationName != "San Pedro" {
		t.Fatalf("copies: %+v", copies)
	}
	if one := expandSlot(slot, booking.AvailableSlotRow{LocationID: sp.LocationID}, byID, active); len(one) != 1 || one[0].LocationName != "San Pedro" || one[0].CustomerChoosesLocation {
		t.Fatalf("fixed: %+v", one)
	}
	if none := expandSlot(slot, booking.AvailableSlotRow{CustomerChoice: true}, byID, nil); len(none) != 1 || none[0].LocationID != "" {
		t.Fatalf("no active location: %+v", none)
	}
	// Any copy can be booked: the request names the place.
	if _, _, l, err := matchSlot(copies, "10:00", "11:00", sp.LocationID); err != nil || l != sp.LocationID {
		t.Fatalf("choose San Pedro: %q %v", l, err)
	}
}

// "Customer chooses" places (2026-11-08): own places in each location when the
// company can attend in several locations at the same time, shared otherwise.
func TestSlotCopiesPlaces(t *testing.T) {
	her := locations.CompanyLocationS{LocationID: "11111111-1111-1111-1111-111111111111", Name: "Heredia", Active: true}
	sp := locations.CompanyLocationS{LocationID: "22222222-2222-2222-2222-222222222222", Name: "San Pedro", Active: true}
	byID := map[string]locations.CompanyLocationS{her.LocationID: her, sp.LocationID: sp}
	active := activeLocations(byID)
	choice := booking.AvailableSlotRow{CustomerChoice: true, Capacity: 3}
	base := AvailableSlot{StartTime: "09:00", EndTime: "10:00", Capacity: 3}
	// Heredia already has 3 (full), San Pedro 1; in any location: 4.
	taken := map[string]int{her.LocationID: 3, sp.LocationID: 1, "": 4}
	count := func(id string) (int, error) { return taken[id], nil }

	own, err := slotCopies(base, choice, byID, active, true, count)
	if err != nil || len(own) != 1 || own[0].LocationName != "San Pedro" || own[0].Remaining != 2 || !own[0].CustomerChoosesLocation {
		t.Fatalf("same time ON: %+v %v", own, err)
	}
	if shared, _ := slotCopies(base, choice, byID, active, false, count); len(shared) != 0 {
		t.Fatalf("same time OFF, 4 taken of 3 shared: %+v", shared)
	}
	taken[""] = 1
	shared, _ := slotCopies(base, choice, byID, active, false, count)
	if len(shared) != 2 || shared[0].Remaining != 2 || shared[1].Remaining != 2 {
		t.Fatalf("same time OFF shared: %+v", shared)
	}
	fixed, _ := slotCopies(base, booking.AvailableSlotRow{LocationID: sp.LocationID, Capacity: 3}, byID, active, true, count)
	if len(fixed) != 1 || fixed[0].LocationName != "San Pedro" || fixed[0].Remaining != 2 {
		t.Fatalf("fixed location: %+v", fixed)
	}
}
