package appointment

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CustomerAppointmentRow is one appointment/booking of a signed-in customer
// ("My bookings").
type CustomerAppointmentRow struct {
	AppointmentID   string    `json:"-"`
	ServiceName     string    `json:"serviceName"`
	AppointmentDate string    `json:"date"`      // YYYY-MM-DD
	StartHour       string    `json:"startHour"` // HH:MM
	EndHour         string    `json:"endHour"`   // HH:MM
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	PublicToken     *string   `json:"-"`
}

// SelectAppointmentsForCustomer returns the appointments of one company
// with the customer's (VERIFIED) email, newest date first.
func SelectAppointmentsForCustomer(companyID, email string, limit int) ([]CustomerAppointmentRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	email = strings.TrimSpace(email)
	if companyID == "" || email == "" {
		return nil, fmt.Errorf("companyId and email are required")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := db.Query(`
		SELECT
			a.appointment_id::text,
			COALESCE(p.name, ''),
			to_char(a.appointment_date, 'YYYY-MM-DD'),
			to_char(a.start_hour, 'HH24:MI'),
			to_char(a.end_hour, 'HH24:MI'),
			a.status,
			a.created_at,
			a.public_token
		FROM public.tbl_appointment a
		LEFT JOIN public.tbl_products p
		       ON p.product_internal_id::text = a.service_id::text
		      AND p.company_id::text = a.company_id::text
		WHERE a.company_id::text = $1
		  AND lower(a.email) = lower($2)
		ORDER BY a.appointment_date DESC, a.start_hour DESC
		LIMIT $3
	`, companyID, email, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select customer appointments: %w", err)
	}
	defer rows.Close()

	list := make([]CustomerAppointmentRow, 0)
	for rows.Next() {
		var r CustomerAppointmentRow
		if err := rows.Scan(
			&r.AppointmentID, &r.ServiceName, &r.AppointmentDate, &r.StartHour,
			&r.EndHour, &r.Status, &r.CreatedAt, &r.PublicToken,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer appointment: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading customer appointments: %w", err)
	}
	return list, nil
}
