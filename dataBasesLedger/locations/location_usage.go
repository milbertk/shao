package locations

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// LocationUsage tells where a location is used.
type LocationUsage struct {
	RecurringRanges    int `json:"recurringRanges"`    // weekly hours of schedules
	ManualSlots        int `json:"manualSlots"`        // hours on specific dates
	Appointments       int `json:"appointments"`       // every appointment (history too)
	FutureAppointments int `json:"futureAppointments"` // from today, not cancelled / rejected / done
}

// Used tells if anything points to the location (then it can only be disabled).
func (u LocationUsage) Used() bool {
	return u.RecurringRanges > 0 || u.ManualSlots > 0 || u.Appointments > 0
}

// CountLocationUsage counts the schedules, slots and appointments that use
// one location of the company. today is the company's date (YYYY-MM-DD).
func CountLocationUsage(companyID, locationID, today string) (LocationUsage, error) {
	var usage LocationUsage
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return usage, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	locationID = strings.TrimSpace(locationID)

	err = db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM public.tbl_booking_recurring_range
			  WHERE company_id::text = $1 AND location_id::text = $2),
			(SELECT COUNT(*) FROM public.tbl_booking_manual_slot
			  WHERE company_id::text = $1 AND location_id::text = $2),
			(SELECT COUNT(*) FROM public.tbl_appointment
			  WHERE company_id::text = $1 AND location_id::text = $2),
			(SELECT COUNT(*) FROM public.tbl_appointment
			  WHERE company_id::text = $1 AND location_id::text = $2
			    AND appointment_date >= $3::date
			    AND status NOT IN ('CANCELLED', 'REJECTED', 'DONE'))
	`, companyID, locationID, strings.TrimSpace(today)).Scan(
		&usage.RecurringRanges,
		&usage.ManualSlots,
		&usage.Appointments,
		&usage.FutureAppointments,
	)
	if err != nil {
		return usage, fmt.Errorf("failed to count location usage: %w", err)
	}
	return usage, nil
}
