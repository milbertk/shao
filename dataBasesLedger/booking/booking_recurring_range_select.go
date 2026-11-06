package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const recurringRangeColumns = `
	recurring_range_id,
	booking_config_id,
	company_id,
	day_of_week,
	to_char(start_time, 'HH24:MI'),
	to_char(end_time, 'HH24:MI'),
	appointments,
	active,
	created_at,
	updated_at
`

func scanRecurringRange(row interface {
	Scan(dest ...any) error
}, r *BookingRecurringRangeS) error {
	return row.Scan(
		&r.RecurringRangeID,
		&r.BookingConfigID,
		&r.CompanyID,
		&r.DayOfWeek,
		&r.StartTime,
		&r.EndTime,
		&r.Appointments,
		&r.Active,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
}

// SelectRecurringRangesByConfig returns every recurring range of a booking
// config, scoped to the company. Ordered by day and start time.
func SelectRecurringRangesByConfig(companyID, bookingConfigID string) ([]BookingRecurringRangeS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	bookingConfigID = strings.TrimSpace(bookingConfigID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if bookingConfigID == "" {
		return nil, fmt.Errorf("bookingConfigId is required")
	}

	query := `
		SELECT ` + recurringRangeColumns + `
		FROM public.tbl_booking_recurring_range
		WHERE booking_config_id = $1
		  AND company_id = $2
		ORDER BY
			CASE day_of_week
				WHEN 'MONDAY' THEN 1
				WHEN 'TUESDAY' THEN 2
				WHEN 'WEDNESDAY' THEN 3
				WHEN 'THURSDAY' THEN 4
				WHEN 'FRIDAY' THEN 5
				WHEN 'SATURDAY' THEN 6
				WHEN 'SUNDAY' THEN 7
				ELSE 8
			END,
			start_time
	`

	rows, err := db.Query(query, bookingConfigID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select recurring ranges: %w", err)
	}
	defer rows.Close()

	list := make([]BookingRecurringRangeS, 0)
	for rows.Next() {
		var item BookingRecurringRangeS
		if err := scanRecurringRange(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan recurring range: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading recurring ranges: %w", err)
	}

	return list, nil
}
