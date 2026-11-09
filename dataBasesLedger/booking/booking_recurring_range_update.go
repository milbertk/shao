package booking

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one recurring range by primary key AND company.
func (a *BookingRecurringRangeS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	recurringRangeID := strings.TrimSpace(a.RecurringRangeID)
	companyID := strings.TrimSpace(a.CompanyID)
	if recurringRangeID == "" {
		return fmt.Errorf("recurringRangeId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_booking_recurring_range
		SET
			day_of_week = $3,
			start_time = $4,
			end_time = $5,
			appointments = $6,
			active = $7,
			location_id = $8,
			customer_chooses_location = $9
		WHERE recurring_range_id = $1
		  AND company_id = $2
		RETURNING recurring_range_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		recurringRangeID,
		companyID,
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
		if err == sql.ErrNoRows {
			return fmt.Errorf("recurring range %q does not exist", recurringRangeID)
		}
		return fmt.Errorf("failed to update recurring range: %w", err)
	}

	return nil
}
