package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// AvailableSlotRow is one raw availability row for a given date, BEFORE
// subtracting the appointments already taken.
type AvailableSlotRow struct {
	Source    string // "RECURRING" or "MANUAL"
	StartTime string // "HH:MM" (empty for all-day)
	EndTime   string // "HH:MM" (empty for all-day)
	AllDay    bool
	Capacity  int // resolved capacity (range value or general fallback)
}

// GetBookingConfigForProduct returns the header fields needed to compute
// availability for a product, scoped to the company. Returns ok=false when the
// product has no (active) booking configuration.
type BookingConfigLite struct {
	BookingConfigID            string
	AvailabilityMode           string // RECURRING | MANUAL
	GeneralAppointments        int
	AppointmentDurationMinutes int
	BookingWindow              string
	Active                     bool
	DisableHolidaysEnabled     bool
	DisabledFrom               string // "YYYY-MM-DD" or ""
	DisabledTo                 string // "YYYY-MM-DD" or ""
}

func GetBookingConfigForProduct(companyID, productID string) (*BookingConfigLite, bool, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, false, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	productID = strings.TrimSpace(productID)
	if companyID == "" || productID == "" {
		return nil, false, fmt.Errorf("companyId and productId are required")
	}

	query := `
		SELECT
			booking_config_id,
			availability_mode,
			general_appointments_per_slot,
			appointment_duration_minutes,
			booking_window,
			active,
			disable_holidays_enabled,
			COALESCE(to_char(disabled_from, 'YYYY-MM-DD'), ''),
			COALESCE(to_char(disabled_to, 'YYYY-MM-DD'), '')
		FROM public.tbl_booking_config
		WHERE company_id = $1
		  AND service_product_id = $2
	`

	var c BookingConfigLite
	err = db.QueryRow(query, companyID, productID).Scan(
		&c.BookingConfigID,
		&c.AvailabilityMode,
		&c.GeneralAppointments,
		&c.AppointmentDurationMinutes,
		&c.BookingWindow,
		&c.Active,
		&c.DisableHolidaysEnabled,
		&c.DisabledFrom,
		&c.DisabledTo,
	)
	if err != nil {
		// No rows -> no booking config for this product.
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to load booking config: %w", err)
	}

	return &c, true, nil
}

// GetRecurringSlotsForWeekday returns the recurring ranges defined for a given
// weekday of a config, with capacity resolved (range value or general).
// weekday must be one of MONDAY..SUNDAY.
func GetRecurringSlotsForWeekday(companyID, bookingConfigID, weekday string, general int) ([]AvailableSlotRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			to_char(start_time, 'HH24:MI'),
			to_char(end_time, 'HH24:MI'),
			COALESCE(appointments, $4)
		FROM public.tbl_booking_recurring_range
		WHERE company_id = $1
		  AND booking_config_id = $2
		  AND day_of_week = $3
		  AND active = TRUE
		ORDER BY start_time
	`

	rows, err := db.Query(query, companyID, bookingConfigID, weekday, general)
	if err != nil {
		return nil, fmt.Errorf("failed to load recurring slots: %w", err)
	}
	defer rows.Close()

	list := make([]AvailableSlotRow, 0)
	for rows.Next() {
		var r AvailableSlotRow
		r.Source = "RECURRING"
		if err := rows.Scan(&r.StartTime, &r.EndTime, &r.Capacity); err != nil {
			return nil, fmt.Errorf("failed to scan recurring slot: %w", err)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// GetManualSlotsForDate returns the manual slots defined for an exact date,
// with capacity resolved (slot value or general).
func GetManualSlotsForDate(companyID, bookingConfigID, date string, general int) ([]AvailableSlotRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			all_day,
			COALESCE(to_char(start_time, 'HH24:MI'), ''),
			COALESCE(to_char(end_time, 'HH24:MI'), ''),
			COALESCE(appointments, $4)
		FROM public.tbl_booking_manual_slot
		WHERE company_id = $1
		  AND booking_config_id = $2
		  AND slot_date = $3
		ORDER BY start_time NULLS FIRST
	`

	rows, err := db.Query(query, companyID, bookingConfigID, date, general)
	if err != nil {
		return nil, fmt.Errorf("failed to load manual slots: %w", err)
	}
	defer rows.Close()

	list := make([]AvailableSlotRow, 0)
	for rows.Next() {
		var r AvailableSlotRow
		r.Source = "MANUAL"
		if err := rows.Scan(&r.AllDay, &r.StartTime, &r.EndTime, &r.Capacity); err != nil {
			return nil, fmt.Errorf("failed to scan manual slot: %w", err)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// CountTakenAppointments returns how many active (not cancelled, not rejected)
// appointments exist for
// a company + product on a given date, overlapping [startTime, endTime).
// For all-day slots pass startTime="00:00", endTime="23:59".
func CountTakenAppointments(companyID, productID, date, startTime, endTime string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT COUNT(*)
		FROM public.tbl_appointment
		WHERE company_id = $1
		  AND service_id = $2
		  AND appointment_date = $3
		  AND status NOT IN ('CANCELLED', 'REJECTED')
		  AND start_hour < $5::time
		  AND $4::time < end_hour
	`

	var count int
	err = db.QueryRow(query, companyID, productID, date, startTime, endTime).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count taken appointments: %w", err)
	}
	return count, nil
}
