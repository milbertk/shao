package appointment

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// EnsurePublicToken returns the appointment's private "manage my request"
// token, saving newToken first when the appointment has none yet.
func EnsurePublicToken(companyID, appointmentID, newToken string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var token string
	err = db.QueryRow(`
		UPDATE public.tbl_appointment
		SET public_token = COALESCE(public_token, $3)
		WHERE appointment_id = $1 AND company_id = $2
		RETURNING public_token
	`, strings.TrimSpace(appointmentID), strings.TrimSpace(companyID), newToken).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to set appointment token: %w", err)
	}
	return token, nil
}

// SelectByPublicToken loads the appointment that owns the token.
// Returns sql.ErrNoRows when no appointment has it.
func SelectByPublicToken(token string) (*AppointmentS, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, sql.ErrNoRows
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var item AppointmentS
	err = db.QueryRow(`
		SELECT appointment_id, company_id
		FROM public.tbl_appointment
		WHERE public_token = $1
	`, token).Scan(&item.AppointmentID, &item.CompanyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to find appointment by token: %w", err)
	}
	if err := item.Select(); err != nil {
		return nil, err
	}
	return &item, nil
}

// MoveToNewTime changes date and hours and sets the status back to PENDING,
// so the business approves the new time again (customer reschedule request).
func MoveToNewTime(companyID, appointmentID, date, startHour, endHour string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	res, err := db.Exec(`
		UPDATE public.tbl_appointment
		SET appointment_date = $3, start_hour = $4, end_hour = $5, status = 'PENDING'
		WHERE appointment_id = $1 AND company_id = $2
	`, strings.TrimSpace(appointmentID), strings.TrimSpace(companyID), date, startHour, endHour)
	if err != nil {
		return fmt.Errorf("failed to move appointment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
