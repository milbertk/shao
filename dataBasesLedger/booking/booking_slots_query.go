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
	// LocationID (2026-11-07): location of the range/slot ("" = none).
	// Ranges and slots of a DISABLED location are not returned while the
	// company uses locations.
	LocationID string
	// CustomerChoice (2026-11-08): offered in any active location; the
	// customer picks it (LocationID is "").
	CustomerChoice bool
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
// weekday must be one of MONDAY..SUNDAY. Ranges of a disabled location are
// skipped (2026-11-07).
//
// locationsOn = the company uses locations (2026-11-08). OFF: every active
// range is returned, whatever its location (as before locations).
func GetRecurringSlotsForWeekday(companyID, bookingConfigID, weekday string, general int, locationsOn bool) ([]AvailableSlotRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			to_char(r.start_time, 'HH24:MI'),
			to_char(r.end_time, 'HH24:MI'),
			COALESCE(r.appointments, $4),
			COALESCE(r.location_id::text, ''),
			r.customer_chooses_location
		FROM public.tbl_booking_recurring_range r
		LEFT JOIN public.tbl_company_location l ON l.location_id = r.location_id
		WHERE r.company_id = $1
		  AND r.booking_config_id = $2
		  AND r.day_of_week = $3
		  AND r.active = TRUE
		  AND ($5 = FALSE OR r.location_id IS NULL OR l.active = TRUE)
		ORDER BY r.start_time
	`

	rows, err := db.Query(query, companyID, bookingConfigID, weekday, general, locationsOn)
	if err != nil {
		return nil, fmt.Errorf("failed to load recurring slots: %w", err)
	}
	defer rows.Close()

	list := make([]AvailableSlotRow, 0)
	for rows.Next() {
		var r AvailableSlotRow
		r.Source = "RECURRING"
		if err := rows.Scan(&r.StartTime, &r.EndTime, &r.Capacity, &r.LocationID, &r.CustomerChoice); err != nil {
			return nil, fmt.Errorf("failed to scan recurring slot: %w", err)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// GetManualSlotsForDate returns the manual slots defined for an exact date,
// with capacity resolved (slot value or general). Slots of a disabled
// location are skipped (2026-11-07).
// locationsOn: see GetRecurringSlotsForWeekday.
func GetManualSlotsForDate(companyID, bookingConfigID, date string, general int, locationsOn bool) ([]AvailableSlotRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			s.all_day,
			COALESCE(to_char(s.start_time, 'HH24:MI'), ''),
			COALESCE(to_char(s.end_time, 'HH24:MI'), ''),
			COALESCE(s.appointments, $4),
			COALESCE(s.location_id::text, ''),
			s.customer_chooses_location
		FROM public.tbl_booking_manual_slot s
		LEFT JOIN public.tbl_company_location l ON l.location_id = s.location_id
		WHERE s.company_id = $1
		  AND s.booking_config_id = $2
		  AND s.slot_date = $3
		  AND ($5 = FALSE OR s.location_id IS NULL OR l.active = TRUE)
		ORDER BY s.start_time NULLS FIRST
	`

	rows, err := db.Query(query, companyID, bookingConfigID, date, general, locationsOn)
	if err != nil {
		return nil, fmt.Errorf("failed to load manual slots: %w", err)
	}
	defer rows.Close()

	list := make([]AvailableSlotRow, 0)
	for rows.Next() {
		var r AvailableSlotRow
		r.Source = "MANUAL"
		if err := rows.Scan(&r.AllDay, &r.StartTime, &r.EndTime, &r.Capacity, &r.LocationID, &r.CustomerChoice); err != nil {
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
//
// Locations (2026-11-07): with locationID, only the appointments in THAT
// location count, plus the ones with no location (older data, counted to be
// safe). With locationID = "" every appointment counts (as before).
func CountTakenAppointments(companyID, productID, date, startTime, endTime, locationID string) (int, error) {
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
		  AND ($6 = '' OR location_id IS NULL OR location_id::text = $6)
	`

	var count int
	err = db.QueryRow(query, companyID, productID, date, startTime, endTime, locationID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count taken appointments: %w", err)
	}
	return count, nil
}
