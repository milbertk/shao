package booking

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one booking config identified by primary key AND company.
func (a *BookingConfigS) Update() error {
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
		UPDATE public.tbl_booking_config
		SET
			service_product_id = $3,
			name = $4,
			appointment_duration_minutes = $5,
			active = $6,
			general_appointments_per_slot = $7,
			booking_window = COALESCE(NULLIF($8, ''), 'ONE_MONTH'),
			disable_holidays_enabled = $9,
			disabled_from = $10,
			disabled_to = $11,
			availability_mode = COALESCE(NULLIF($12, ''), 'RECURRING')
		WHERE booking_config_id = $1
		  AND company_id = $2
		RETURNING booking_config_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		bookingConfigID,
		companyID,
		a.ServiceProductID,
		a.Name,
		a.AppointmentDurationMinutes,
		a.Active,
		a.GeneralAppointmentsPerSlot,
		a.BookingWindow,
		a.DisableHolidaysEnabled,
		a.DisabledFrom,
		a.DisabledTo,
		a.AvailabilityMode,
	).Scan(
		&a.BookingConfigID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("booking config %q does not exist", bookingConfigID)
		}
		return fmt.Errorf("failed to update booking config: %w", err)
	}

	return nil
}
