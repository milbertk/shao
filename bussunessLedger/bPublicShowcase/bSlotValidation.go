package blpublicshowcase

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	blproducts "github.com/milbertk/shao/bussunessLedger/products"
)

// ErrSlotTaken is returned when the chosen time is no longer free (or never
// was). Handlers answer 409 with code SLOT_TAKEN so the store can reload the
// available times.
var ErrSlotTaken = errors.New("SLOT_TAKEN: this time was just taken, please choose another")

// ErrServiceNotBookable is returned when the service cannot be booked online.
var ErrServiceNotBookable = errors.New("invalid service: it does not accept online bookings")

const allDayStart, allDayEnd = "00:00", "23:59"

// ValidateRequestedSlot checks on the server that a public booking request is
// valid and returns the hours to store:
//   - the service belongs to the company, is ACTIVE and shown on the web;
//   - the date is valid, not in the past and inside the booking window
//     (rules of BGetAvailabilityForDate);
//   - the time matches one of the offered slots that still has capacity.
//
// An all-day slot is stored as 00:00–23:59.
func ValidateRequestedSlot(companyID, serviceID, date, startHour, endHour string) (string, string, error) {
	product, err := blproducts.BSelectProductByInternalID(companyID, serviceID)
	if err != nil {
		if errors.Is(err, blproducts.ErrProductNotFound) {
			return "", "", ErrServiceNotBookable
		}
		return "", "", err
	}
	if !product.ShowOnWeb || !strings.EqualFold(product.Status, "ACTIVE") {
		return "", "", ErrServiceNotBookable
	}

	if _, err := time.Parse(dateLayout, date); err != nil {
		return "", "", fmt.Errorf("invalid appointmentDate (use YYYY-MM-DD)")
	}

	availability, err := BGetAvailabilityForDate(companyID, serviceID, date)
	if err != nil {
		return "", "", err
	}
	if !availability.Available {
		if availability.Reason == "no active booking configuration" {
			return "", "", ErrServiceNotBookable
		}
		return "", "", ErrSlotTaken
	}

	start, end := hhmm(startHour), hhmm(endHour)
	for _, slot := range availability.Slots {
		if slot.Remaining <= 0 {
			continue
		}
		if slot.AllDay {
			if (start == "" && end == "") || (start == allDayStart && end == allDayEnd) {
				return allDayStart, allDayEnd, nil
			}
			continue
		}
		if slot.StartTime == start && slot.EndTime == end {
			return start, end, nil
		}
	}
	return "", "", ErrSlotTaken
}

// hhmm normalizes "9:00", "09:00" or "09:00:00" to "09:00".
func hhmm(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, layout := range []string{"15:04", "15:04:05", "3:04"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format("15:04")
		}
	}
	return value
}

// Slot locks: one lock per company + service + date while a request is being
// validated and saved. In-memory (one server). With several servers use a
// database advisory lock instead.
var (
	slotLocksMu sync.Mutex
	slotLocks   = map[string]*slotLock{}
)

type slotLock struct {
	mu    sync.Mutex
	users int
}

func lockSlot(companyID, serviceID, date string) func() {
	key := companyID + "|" + serviceID + "|" + date

	slotLocksMu.Lock()
	l, ok := slotLocks[key]
	if !ok {
		l = &slotLock{}
		slotLocks[key] = l
	}
	l.users++
	slotLocksMu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		slotLocksMu.Lock()
		l.users--
		if l.users == 0 {
			delete(slotLocks, key)
		}
		slotLocksMu.Unlock()
	}
}
