package tracking

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteStepsByConfig removes ALL steps of a tracking header, scoped to the
// company. Used by the "replace all" flow: on every save the steps are wiped
// and re-inserted, which keeps step_order consistent.
//
// Returns how many rows were deleted.
func DeleteStepsByConfig(companyID, trackingConfigID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	trackingConfigID = strings.TrimSpace(trackingConfigID)
	if companyID == "" {
		return 0, fmt.Errorf("companyId is required")
	}
	if trackingConfigID == "" {
		return 0, fmt.Errorf("trackingConfigId is required")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_tracking_step
		WHERE tracking_config_id = $1
		  AND company_id = $2
	`, trackingConfigID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete tracking steps: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted tracking steps: %w", err)
	}

	return deleted, nil
}
