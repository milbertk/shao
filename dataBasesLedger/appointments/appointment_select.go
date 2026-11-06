package appointment

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const appointmentColumns = `
	appointment_id,
	company_id,
	service_id,
	use_registered_customer,
	customer_id,
	customer_name,
	phone,
	email,
	social_link,
	to_char(appointment_date, 'YYYY-MM-DD'),
	to_char(start_hour, 'HH24:MI'),
	to_char(end_hour, 'HH24:MI'),
	status,
	comments,
	registered_by,
	created_at,
	updated_at
`

func scanAppointment(row interface {
	Scan(dest ...any) error
}, a *AppointmentS) error {
	return row.Scan(
		&a.AppointmentID,
		&a.CompanyID,
		&a.ServiceID,
		&a.UseRegisteredCustomer,
		&a.CustomerID,
		&a.CustomerName,
		&a.Phone,
		&a.Email,
		&a.SocialLink,
		&a.AppointmentDate,
		&a.StartHour,
		&a.EndHour,
		&a.Status,
		&a.Comments,
		&a.RegisteredBy,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
}

// Select loads one appointment by primary key, scoped to its company.
func (a *AppointmentS) Select() error {
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
		SELECT ` + appointmentColumns + `
		FROM public.tbl_appointment
		WHERE appointment_id = $1
		  AND company_id = $2
	`

	if err := scanAppointment(db.QueryRow(query, appointmentID, companyID), a); err != nil {
		return fmt.Errorf("failed to select appointment: %w", err)
	}

	return nil
}

// CountConflicts returns how many appointments of the same company, service and
// date overlap the given [startHour, endHour). An optional excludeID skips a
// specific appointment (used when updating). Overlap: start < otherEnd AND
// otherStart < end.
func CountConflicts(companyID, serviceID, date, startHour, endHour, excludeID string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT COUNT(*)
		FROM public.tbl_appointment
		WHERE company_id = $1
		  AND service_id = $2
		  AND appointment_date = $3
		  AND status NOT IN ('CANCELLED', 'REJECTED')
		  AND start_hour < $5::time
		  AND $4::time < end_hour
		  AND ($6 = '' OR appointment_id <> $6::uuid)
	`

	var count int
	err = db.QueryRow(
		query,
		strings.TrimSpace(companyID),
		strings.TrimSpace(serviceID),
		strings.TrimSpace(date),
		strings.TrimSpace(startHour),
		strings.TrimSpace(endHour),
		strings.TrimSpace(excludeID),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count appointment conflicts: %w", err)
	}

	return count, nil
}
