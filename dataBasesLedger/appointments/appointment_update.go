package appointment

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one appointment by primary key AND company.
func (a *AppointmentS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	appointmentID := strings.TrimSpace(a.AppointmentID)
	companyID := strings.TrimSpace(a.CompanyID)
	if appointmentID == "" {
		return fmt.Errorf("appointmentId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_appointment
		SET
			service_id = $3,
			use_registered_customer = $4,
			customer_id = $5,
			customer_name = $6,
			phone = $7,
			email = $8,
			social_link = $9,
			appointment_date = $10,
			start_hour = $11,
			end_hour = $12,
			status = COALESCE(NULLIF($13, ''), 'SCHEDULED'),
			comments = $14,
			registered_by = $15
		WHERE appointment_id = $1
		  AND company_id = $2
		RETURNING appointment_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		appointmentID,
		companyID,
		a.ServiceID,
		a.UseRegisteredCustomer,
		a.CustomerID,
		a.CustomerName,
		a.Phone,
		a.Email,
		a.SocialLink,
		a.AppointmentDate,
		a.StartHour,
		a.EndHour,
		a.Status,
		a.Comments,
		a.RegisteredBy,
	).Scan(
		&a.AppointmentID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("appointment %q does not exist", appointmentID)
		}
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	return nil
}
