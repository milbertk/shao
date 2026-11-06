package customers

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one customer identified by primary key (CustomerInternalID)
// AND company (CompanyID), so a customer of another company can never be
// updated. updated_at is refreshed by the database trigger.
func (a *CustomerS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	customerInternalID := strings.TrimSpace(a.CustomerInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	if customerInternalID == "" {
		return fmt.Errorf("customerInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_customers
		SET
			customer_id = $3,
			name = $4,
			alias = $5,
			cellphone = $6,
			social_network = COALESCE(NULLIF($7, ''), 'WHATSAPP'),
			social_network_link = $8,
			birthday = $9,
			birth_year = $10,
			email = $11,
			address = $12,
			website = $13,
			status = COALESCE(NULLIF($14, ''), 'ACTIVE'),
			category = COALESCE(NULLIF($15, ''), 'A'),
			preferred_contact_method = COALESCE(NULLIF($16, ''), 'WHATSAPP'),
			comments = $17
		WHERE customer_internal_id = $1
		  AND company_id = $2
		RETURNING customer_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		customerInternalID,
		companyID,
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
		if err == sql.ErrNoRows {
			return fmt.Errorf("customer %q does not exist", customerInternalID)
		}
		return fmt.Errorf("failed to update customer: %w", err)
	}

	return nil
}
