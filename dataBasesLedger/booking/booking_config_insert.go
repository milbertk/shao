package booking

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// BookingConfigS represents one row from public.tbl_booking_config.
type BookingConfigS struct {
	BookingConfigID             string    `json:"bookingConfigId"`
	CompanyID                   string    `json:"companyId"`
	ServiceProductID            string    `json:"serviceProductId"`
	Name                        string    `json:"name"`
	AppointmentDurationMinutes  int       `json:"appointmentDurationMinutes"`
	Active                      bool      `json:"active"`
	GeneralAppointmentsPerSlot  int       `json:"generalAppointmentsPerSlot"`
	BookingWindow               string    `json:"bookingWindow"`
	DisableHolidaysEnabled      bool      `json:"disableHolidaysEnabled"`
	DisabledFrom                *string   `json:"disabledFrom,omitempty"` // "YYYY-MM-DD"
	DisabledTo                  *string   `json:"disabledTo,omitempty"`   // "YYYY-MM-DD"
	AvailabilityMode            string    `json:"availabilityMode"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

// Insert creates one booking configuration (header) for a company.
func (a *BookingConfigS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_booking_config (
			company_id,
			service_product_id,
			name,
			appointment_duration_minutes,
			active,
			general_appointments_per_slot,
			booking_window,
			disable_holidays_enabled,
			disabled_from,
			disabled_to,
			availability_mode
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			COALESCE(NULLIF($7, ''), 'ONE_MONTH'),
			$8, $9, $10,
			COALESCE(NULLIF($11, ''), 'RECURRING')
		)
		RETURNING booking_config_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
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
		return fmt.Errorf("failed to insert booking config: %w", err)
	}

	return nil
}
