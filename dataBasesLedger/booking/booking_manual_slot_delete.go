package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteManualSlotsByConfig removes ALL manual slots of a booking config
// (used for the "replace all" flow). Returns how many were deleted.
func DeleteManualSlotsByConfig(companyID, bookingConfigID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	bookingConfigID = strings.TrimSpace(bookingConfigID)
	if companyID == "" {
		return 0, fmt.Errorf("companyId is required")
	}
	if bookingConfigID == "" {
		return 0, fmt.Errorf("bookingConfigId is required")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_booking_manual_slot
		WHERE booking_config_id = $1
		  AND company_id = $2
	`, bookingConfigID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete manual slots: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted manual slots: %w", err)
	}

	return deleted, nil
}
