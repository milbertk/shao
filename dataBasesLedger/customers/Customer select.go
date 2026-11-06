package customers

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const customerSelectColumns = `
	customer_internal_id,
	company_id,
	customer_id,
	name,
	alias,
	cellphone,
	social_network,
	social_network_link,
	to_char(birthday, 'YYYY-MM-DD'),
	birth_year,
	email,
	address,
	website,
	status,
	category,
	preferred_contact_method,
	comments,
	created_at,
	updated_at
`

// scanCustomer scans one row (in the customerSelectColumns order) into c.
func scanCustomer(row interface {
	Scan(dest ...any) error
}, c *CustomerS) error {
	return row.Scan(
		&c.CustomerInternalID,
		&c.CompanyID,
		&c.CustomerID,
		&c.Name,
		&c.Alias,
		&c.Cellphone,
		&c.SocialNetwork,
		&c.SocialNetworkLink,
		&c.Birthday,
		&c.BirthYear,
		&c.Email,
		&c.Address,
		&c.Website,
		&c.Status,
		&c.Category,
		&c.PreferredContactMethod,
		&c.Comments,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
}

// Select loads one customer by primary key (CustomerInternalID), scoped to its
// company. Set CustomerInternalID and CompanyID before calling. Returns
// sql.ErrNoRows (wrapped) when not found in that company.
func (a *CustomerS) Select() error {
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
		SELECT ` + customerSelectColumns + `
		FROM public.tbl_customers
		WHERE customer_internal_id = $1
		  AND company_id = $2
	`

	if err := scanCustomer(db.QueryRow(query, customerInternalID, companyID), a); err != nil {
		return fmt.Errorf("failed to select customer: %w", err)
	}

	return nil
}
