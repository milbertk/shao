package appointment

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// UpdateAppointmentStatus changes ONLY the status of an appointment (company
// scoped). Returns rows affected. Used by approve/reject so other fields are
// never touched.
func UpdateAppointmentStatus(companyID, appointmentID, status string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	res, err := db.Exec(`
		UPDATE public.tbl_appointment
		SET status = $3
		WHERE appointment_id = $1 AND company_id = $2
	`, strings.TrimSpace(appointmentID), strings.TrimSpace(companyID), strings.TrimSpace(status))
	if err != nil {
		return 0, fmt.Errorf("failed to update appointment status: %w", err)
	}
	return res.RowsAffected()
}

// GetAppointmentStatus returns the current status of an appointment (or
// sql.ErrNoRows when it does not exist). Used to block double-approval.
func GetAppointmentStatus(companyID, appointmentID string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var status string
	err = db.QueryRow(`
		SELECT status FROM public.tbl_appointment
		WHERE appointment_id = $1 AND company_id = $2
	`, strings.TrimSpace(appointmentID), strings.TrimSpace(companyID)).Scan(&status)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("failed to read appointment status: %w", err)
	}
	return status, nil
}
