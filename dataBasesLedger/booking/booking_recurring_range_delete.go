package booking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteRecurringRangesByConfig removes ALL recurring ranges of a booking
// config (used for the "replace all" flow). Returns how many were deleted.
func DeleteRecurringRangesByConfig(companyID, bookingConfigID string) (int64, error) {
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
		DELETE FROM public.tbl_booking_recurring_range
		WHERE booking_config_id = $1
		  AND company_id = $2
	`, bookingConfigID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete recurring ranges: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted recurring ranges: %w", err)
	}

	return deleted, nil
}
