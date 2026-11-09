package booking

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// BookingRecurringRangeS represents one row from tbl_booking_recurring_range.
type BookingRecurringRangeS struct {
	RecurringRangeID string    `json:"recurringRangeId"`
	BookingConfigID  string    `json:"bookingConfigId"`
	CompanyID        string    `json:"companyId"`
	DayOfWeek        string    `json:"dayOfWeek"`
	StartTime        string    `json:"startTime"` // "HH:MM"
	EndTime          string    `json:"endTime"`   // "HH:MM"
	Appointments     *int      `json:"appointments,omitempty"`
	Active           bool      `json:"active"`
	LocationID       *string   `json:"locationId,omitempty"`    // optional location of these hours (2026-11-07); nil = none
	CustomerChoice   bool      `json:"customerChoosesLocation"` // any active location, the customer picks it (2026-11-08)
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Insert creates one recurring range for a booking config.
func (a *BookingRecurringRangeS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_booking_recurring_range (
			booking_config_id,
			company_id,
			day_of_week,
			start_time,
			end_time,
			appointments,
			active,
			location_id,
			customer_chooses_location
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING recurring_range_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.BookingConfigID,
		a.CompanyID,
		a.DayOfWeek,
		a.StartTime,
		a.EndTime,
		a.Appointments,
		a.Active,
		a.LocationID,
		a.CustomerChoice,
	).Scan(
		&a.RecurringRangeID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert recurring range: %w", err)
	}

	return nil
}
