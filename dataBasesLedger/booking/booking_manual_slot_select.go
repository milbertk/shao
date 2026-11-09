package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const manualSlotColumns = `
	manual_slot_id,
	booking_config_id,
	company_id,
	to_char(slot_date, 'YYYY-MM-DD'),
	all_day,
	to_char(start_time, 'HH24:MI'),
	to_char(end_time, 'HH24:MI'),
	appointments,
	created_at,
	updated_at,
	location_id::text,
	customer_chooses_location
`

func scanManualSlot(row interface {
	Scan(dest ...any) error
}, s *BookingManualSlotS) error {
	return row.Scan(
		&s.ManualSlotID,
		&s.BookingConfigID,
		&s.CompanyID,
		&s.SlotDate,
		&s.AllDay,
		&s.StartTime,
		&s.EndTime,
		&s.Appointments,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.LocationID,
		&s.CustomerChoice,
	)
}

// SelectManualSlotsByConfig returns every manual slot of a booking config,
// scoped to the company. Ordered by date and start time.
func SelectManualSlotsByConfig(companyID, bookingConfigID string) ([]BookingManualSlotS, error) {
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
		SELECT ` + manualSlotColumns + `
		FROM public.tbl_booking_manual_slot
		WHERE booking_config_id = $1
		  AND company_id = $2
		ORDER BY slot_date, start_time NULLS FIRST
	`

	rows, err := db.Query(query, bookingConfigID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select manual slots: %w", err)
	}
	defer rows.Close()

	list := make([]BookingManualSlotS, 0)
	for rows.Next() {
		var item BookingManualSlotS
		if err := scanManualSlot(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan manual slot: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading manual slots: %w", err)
	}

	return list, nil
}
