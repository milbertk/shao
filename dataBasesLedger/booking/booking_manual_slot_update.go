package booking

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one manual slot by primary key AND company.
func (a *BookingManualSlotS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	manualSlotID := strings.TrimSpace(a.ManualSlotID)
	companyID := strings.TrimSpace(a.CompanyID)
	if manualSlotID == "" {
		return fmt.Errorf("manualSlotId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_booking_manual_slot
		SET
			slot_date = $3,
			all_day = $4,
			start_time = $5,
			end_time = $6,
			appointments = $7
		WHERE manual_slot_id = $1
		  AND company_id = $2
		RETURNING manual_slot_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		manualSlotID,
		companyID,
		a.SlotDate,
		a.AllDay,
		a.StartTime,
		a.EndTime,
		a.Appointments,
	).Scan(
		&a.ManualSlotID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("manual slot %q does not exist", manualSlotID)
		}
		return fmt.Errorf("failed to update manual slot: %w", err)
	}

	return nil
}
