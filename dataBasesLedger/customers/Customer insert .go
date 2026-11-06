package customers

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CustomerS represents one row from public.tbl_customers.
type CustomerS struct {
	CustomerInternalID     string    `json:"customerInternalId"`
	CompanyID              string    `json:"companyId"`
	CustomerID             string    `json:"customerId"`
	Name                   string    `json:"name"`
	Alias                  *string   `json:"alias,omitempty"`
	Cellphone              *string   `json:"cellphone,omitempty"`
	SocialNetwork          string    `json:"socialNetwork"`
	SocialNetworkLink      *string   `json:"socialNetworkLink,omitempty"`
	Birthday               *string   `json:"birthday,omitempty"` // "YYYY-MM-DD"
	BirthYear              *int      `json:"birthYear,omitempty"`
	Email                  *string   `json:"email,omitempty"`
	Address                *string   `json:"address,omitempty"`
	Website                *string   `json:"website,omitempty"`
	Status                 string    `json:"status"`
	Category               string    `json:"category"`
	PreferredContactMethod string    `json:"preferredContactMethod"`
	Comments               *string   `json:"comments,omitempty"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

// Insert creates one customer for a company. company_id is mandatory. Columns
// with a database DEFAULT (social_network, status, category,
// preferred_contact_method) fall back to the default when sent empty.
func (a *CustomerS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_customers (
			company_id,
			customer_id,
			name,
			alias,
			cellphone,
			social_network,
			social_network_link,
			birthday,
			birth_year,
			email,
			address,
			website,
			status,
			category,
			preferred_contact_method,
			comments
		)
		VALUES (
			$1, $2, $3, $4, $5,
			COALESCE(NULLIF($6, ''), 'WHATSAPP'),
			$7, $8, $9, $10, $11, $12,
			COALESCE(NULLIF($13, ''), 'ACTIVE'),
			COALESCE(NULLIF($14, ''), 'A'),
			COALESCE(NULLIF($15, ''), 'WHATSAPP'),
			$16
		)
		RETURNING customer_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
		a.CustomerID,
		a.Name,
		a.Alias,
		a.Cellphone,
		a.SocialNetwork,
		a.SocialNetworkLink,
		a.Birthday,
		a.BirthYear,
		a.Email,
		a.Address,
		a.Website,
		a.Status,
		a.Category,
		a.PreferredContactMethod,
		a.Comments,
	).Scan(
		&a.CustomerInternalID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert customer: %w", err)
	}

	return nil
}
