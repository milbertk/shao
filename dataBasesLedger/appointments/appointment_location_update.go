package appointment

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// UpdateAppointmentLocation changes ONLY the location of an appointment
// (company scoped; nil = no location). Returns rows affected. 2026-11-08.
func UpdateAppointmentLocation(companyID, appointmentID string, locationID *string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	res, err := db.Exec(`
		UPDATE public.tbl_appointment
		SET location_id = $3
		WHERE appointment_id = $1 AND company_id = $2
	`, strings.TrimSpace(appointmentID), strings.TrimSpace(companyID), locationID)
	if err != nil {
		return 0, fmt.Errorf("failed to update appointment location: %w", err)
	}
	return res.RowsAffected()
}
