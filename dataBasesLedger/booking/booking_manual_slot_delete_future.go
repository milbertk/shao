package booking

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteManualSlotsFromTodayByConfig removes the manual slots of a booking
// config whose date is TODAY or in the FUTURE, scoped to the company. Past
// slots are kept as history. Used by the "replace all (refresh)" flow so
// rewriting future availability never destroys past records.
//
// today is the COMPANY's date (YYYY-MM-DD, its time zone, 2026-11-04); the
// database's CURRENT_DATE would be the server's (UTC) date.
//
// Returns how many rows were deleted.
func DeleteManualSlotsFromTodayByConfig(companyID, bookingConfigID, today string) (int64, error) {
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
	today = strings.TrimSpace(today)
	if _, err := time.Parse("2006-01-02", today); err != nil {
		return 0, fmt.Errorf("invalid today date (use YYYY-MM-DD)")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_booking_manual_slot
		WHERE booking_config_id = $1
		  AND company_id = $2
		  AND slot_date >= $3::date
	`, bookingConfigID, companyID, today)
	if err != nil {
		return 0, fmt.Errorf("failed to delete future manual slots: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted manual slots: %w", err)
	}

	return deleted, nil
}
