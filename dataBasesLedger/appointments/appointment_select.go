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
	updated_at,
	location_id::text,
	(SELECT l.name FROM public.tbl_company_location l WHERE l.location_id = tbl_appointment.location_id),
	(SELECT l.address FROM public.tbl_company_location l WHERE l.location_id = tbl_appointment.location_id),
	(SELECT l.latitude FROM public.tbl_company_location l WHERE l.location_id = tbl_appointment.location_id),
	(SELECT l.longitude FROM public.tbl_company_location l WHERE l.location_id = tbl_appointment.location_id)
`

// appointmentColumns needs "FROM public.tbl_appointment" WITHOUT an alias
// (the location columns, 2026-11-07, refer to tbl_appointment.location_id).

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
		&a.LocationID,
		&a.LocationName,
		&a.LocationAddress,
		&a.LocationLatitude,
		&a.LocationLongitude,
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
//
// Locations (2026-11-07): with locationID, only the appointments in THAT
// location (or with no location) count; with locationID = "" all count.
func CountConflicts(companyID, serviceID, date, startHour, endHour, excludeID, locationID string) (int, error) {
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
		  AND ($7 = '' OR location_id IS NULL OR location_id::text = $7)
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
		strings.TrimSpace(locationID),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count appointment conflicts: %w", err)
	}

	return count, nil
}

// CountOtherLocationConflicts returns how many active appointments of the
// company (ANY service) overlap [startHour, endHour) on the date in ANOTHER
// location (2026-11-07). Used when the company attends in one location at a
// time: the person can not be in two places. Appointments with no location
// are not counted. excludeID skips the appointment being saved.
func CountOtherLocationConflicts(companyID, date, startHour, endHour, locationID, excludeID string) (int, error) {
	locationID = strings.TrimSpace(locationID)
	if locationID == "" {
		return 0, nil
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_appointment
		WHERE company_id = $1
		  AND appointment_date = $2
		  AND status NOT IN ('CANCELLED', 'REJECTED')
		  AND start_hour < $4::time
		  AND $3::time < end_hour
		  AND location_id IS NOT NULL
		  AND location_id::text <> $5
		  AND ($6 = '' OR appointment_id <> $6::uuid)
	`,
		strings.TrimSpace(companyID),
		strings.TrimSpace(date),
		strings.TrimSpace(startHour),
		strings.TrimSpace(endHour),
		locationID,
		strings.TrimSpace(excludeID),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count appointments in other locations: %w", err)
	}
	return count, nil
}
