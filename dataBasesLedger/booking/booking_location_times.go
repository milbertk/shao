package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// LocationTimeRow is one block of hours WITH a location, of a schedule that
// is in use (2026-11-07). Used to check that a company that attends in one
// location at a time does not get the same hours in two locations.
type LocationTimeRow struct {
	Kind            string // RECURRING | MANUAL
	BookingConfigID string
	ScheduleName    string
	DayOfWeek       string // RECURRING: MONDAY..SUNDAY
	SlotDate        string // MANUAL: YYYY-MM-DD
	AllDay          bool   // MANUAL only
	StartTime       string // HH:MM ("" for all-day)
	EndTime         string // HH:MM ("" for all-day)
	LocationID      string // "*" = customer chooses (any active location)
	LocationName    string
}

// SelectCompanyLocationTimes returns the hours with a location of every
// ACTIVE schedule of the company, except excludeConfigID (the schedule being
// saved):
//   - weekly ranges (active) of schedules in RECURRING mode;
//   - manual slots from fromDate (company's today) of schedules in MANUAL mode.
//
// Hours of a disabled location are not returned (they are not offered),
// except the ones of alsoLocationID (a location that is being enabled).
// Hours where the customer chooses the location (2026-11-08) come with
// LocationID "*".
func SelectCompanyLocationTimes(companyID, excludeConfigID, fromDate, alsoLocationID string) ([]LocationTimeRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	query := `
		SELECT
			'RECURRING',
			c.booking_config_id::text,
			COALESCE(c.name, ''),
			r.day_of_week::text,
			'',
			FALSE,
			to_char(r.start_time, 'HH24:MI'),
			to_char(r.end_time, 'HH24:MI'),
			CASE WHEN r.customer_chooses_location THEN '*' ELSE r.location_id::text END,
			COALESCE(l.name, '')
		FROM public.tbl_booking_recurring_range r
		JOIN public.tbl_booking_config c ON c.booking_config_id = r.booking_config_id
		LEFT JOIN public.tbl_company_location l ON l.location_id = r.location_id
		WHERE r.company_id::text = $1
		  AND r.active = TRUE
		  AND c.active = TRUE
		  AND (r.customer_chooses_location
		       OR (r.location_id IS NOT NULL AND (l.active = TRUE OR l.location_id::text = $4)))
		  AND upper(c.availability_mode::text) <> 'MANUAL'
		  AND ($2 = '' OR c.booking_config_id::text <> $2)

		UNION ALL

		SELECT
			'MANUAL',
			c.booking_config_id::text,
			COALESCE(c.name, ''),
			'',
			to_char(s.slot_date, 'YYYY-MM-DD'),
			s.all_day,
			COALESCE(to_char(s.start_time, 'HH24:MI'), ''),
			COALESCE(to_char(s.end_time, 'HH24:MI'), ''),
			CASE WHEN s.customer_chooses_location THEN '*' ELSE s.location_id::text END,
			COALESCE(l.name, '')
		FROM public.tbl_booking_manual_slot s
		JOIN public.tbl_booking_config c ON c.booking_config_id = s.booking_config_id
		LEFT JOIN public.tbl_company_location l ON l.location_id = s.location_id
		WHERE s.company_id::text = $1
		  AND c.active = TRUE
		  AND (s.customer_chooses_location
		       OR (s.location_id IS NOT NULL AND (l.active = TRUE OR l.location_id::text = $4)))
		  AND upper(c.availability_mode::text) = 'MANUAL'
		  AND s.slot_date >= $3::date
		  AND ($2 = '' OR c.booking_config_id::text <> $2)
	`

	rows, err := db.Query(query, companyID, strings.TrimSpace(excludeConfigID), strings.TrimSpace(fromDate), strings.TrimSpace(alsoLocationID))
	if err != nil {
		return nil, fmt.Errorf("failed to load location hours: %w", err)
	}
	defer rows.Close()

	list := make([]LocationTimeRow, 0)
	for rows.Next() {
		var r LocationTimeRow
		if err := rows.Scan(
			&r.Kind,
			&r.BookingConfigID,
			&r.ScheduleName,
			&r.DayOfWeek,
			&r.SlotDate,
			&r.AllDay,
			&r.StartTime,
			&r.EndTime,
			&r.LocationID,
			&r.LocationName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan location hours: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading location hours: %w", err)
	}
	return list, nil
}
