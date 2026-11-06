package appointment

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Delete removes one appointment by primary key AND company (hard delete).
// Returns how many rows were removed (0 if it did not exist in that company).
func Delete(companyID, appointmentID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	appointmentID = strings.TrimSpace(appointmentID)
	if companyID == "" {
		return 0, fmt.Errorf("companyId is required")
	}
	if appointmentID == "" {
		return 0, fmt.Errorf("appointmentId is required")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_appointment
		WHERE appointment_id = $1
		  AND company_id = $2
	`, appointmentID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete appointment: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted appointment: %w", err)
	}

	return deleted, nil
}
