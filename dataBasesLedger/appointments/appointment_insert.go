package appointment

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// AppointmentS represents one row from public.tbl_appointment.
type AppointmentS struct {
	AppointmentID         string    `json:"appointmentId"`
	CompanyID             string    `json:"companyId"`
	ServiceID             string    `json:"serviceId"`
	UseRegisteredCustomer bool      `json:"useRegisteredCustomer"`
	CustomerID            *string   `json:"customerId,omitempty"`
	CustomerName          string    `json:"customerName"`
	Phone                 *string   `json:"phone,omitempty"`
	Email                 *string   `json:"email,omitempty"`
	SocialLink            *string   `json:"socialLink,omitempty"`
	AppointmentDate       string    `json:"appointmentDate"` // YYYY-MM-DD
	StartHour             string    `json:"startHour"`       // HH:MM
	EndHour               string    `json:"endHour"`         // HH:MM
	Status                string    `json:"status"`
	Comments              *string   `json:"comments,omitempty"`
	RegisteredBy          *string   `json:"registeredBy,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// Insert creates one appointment.
func (a *AppointmentS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_appointment (
			company_id,
			service_id,
			use_registered_customer,
			customer_id,
			customer_name,
			phone,
			email,
			social_link,
			appointment_date,
			start_hour,
			end_hour,
			status,
			comments,
			registered_by
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			COALESCE(NULLIF($12, ''), 'SCHEDULED'),
			$13, $14
		)
		RETURNING appointment_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
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
		return fmt.Errorf("failed to insert appointment: %w", err)
	}

	return nil
}
