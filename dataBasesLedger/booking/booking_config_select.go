package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const bookingConfigColumns = `
	booking_config_id,
	company_id,
	service_product_id,
	name,
	appointment_duration_minutes,
	active,
	general_appointments_per_slot,
	booking_window,
	disable_holidays_enabled,
	to_char(disabled_from, 'YYYY-MM-DD'),
	to_char(disabled_to, 'YYYY-MM-DD'),
	availability_mode,
	created_at,
	updated_at
`

func scanBookingConfig(row interface {
	Scan(dest ...any) error
}, c *BookingConfigS) error {
	return row.Scan(
		&c.BookingConfigID,
		&c.CompanyID,
		&c.ServiceProductID,
		&c.Name,
		&c.AppointmentDurationMinutes,
		&c.Active,
		&c.GeneralAppointmentsPerSlot,
		&c.BookingWindow,
		&c.DisableHolidaysEnabled,
		&c.DisabledFrom,
		&c.DisabledTo,
		&c.AvailabilityMode,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
}

// Select loads one booking config by primary key, scoped to its company.
func (a *BookingConfigS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	bookingConfigID := strings.TrimSpace(a.BookingConfigID)
	companyID := strings.TrimSpace(a.CompanyID)
	if bookingConfigID == "" {
		return fmt.Errorf("bookingConfigId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		SELECT ` + bookingConfigColumns + `
		FROM public.tbl_booking_config
		WHERE booking_config_id = $1
		  AND company_id = $2
	`

	if err := scanBookingConfig(db.QueryRow(query, bookingConfigID, companyID), a); err != nil {
		return fmt.Errorf("failed to select booking config: %w", err)
	}

	return nil
}
