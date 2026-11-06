package customers

import (
	"fmt"
	"strconv"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CustomerSearchFilters holds optional search criteria. CompanyID is REQUIRED
// and always applied, so a search never crosses company boundaries.
type CustomerSearchFilters struct {
	CompanyID         string
	Term              string
	CustomerID        string
	Name              string
	Alias             string
	Email             string
	Cellphone         string
	SocialNetworkLink string
	SocialNetwork     string
	Status            string
	Category          string
	Limit             int
	Offset            int
}

// SearchCustomers returns the customers of ONE company that match the filters,
// ordered by name.
func SearchCustomers(filters CustomerSearchFilters) ([]CustomerS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(filters.CompanyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	// company_id is always the first condition.
	conditions := []string{"company_id = $1"}
	args := []interface{}{companyID}
	position := 2

	addLike := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", column, position))
		args = append(args, "%"+value+"%")
		position++
	}

	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, position))
		args = append(args, value)
		position++
	}

	term := strings.TrimSpace(filters.Term)
	if term != "" {
		conditions = append(conditions, fmt.Sprintf(
			`(
				customer_id ILIKE $%d
				OR name ILIKE $%d
				OR COALESCE(alias, '') ILIKE $%d
				OR COALESCE(email, '') ILIKE $%d
				OR COALESCE(cellphone, '') ILIKE $%d
				OR COALESCE(social_network_link, '') ILIKE $%d
			)`,
			position, position, position, position, position, position,
		))
		args = append(args, "%"+term+"%")
		position++
	}

	addLike("customer_id", filters.CustomerID)
	addLike("name", filters.Name)
	addLike("COALESCE(alias, '')", filters.Alias)
	addLike("COALESCE(email, '')", filters.Email)
	addLike("COALESCE(cellphone, '')", filters.Cellphone)
	addLike("COALESCE(social_network_link, '')", filters.SocialNetworkLink)

	addEqual("social_network", strings.ToUpper(filters.SocialNetwork))
	addEqual("status", strings.ToUpper(filters.Status))
	addEqual("category", strings.ToUpper(filters.Category))

	whereClause := "WHERE " + strings.Join(conditions, "\n\t\t\t  AND ")

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	limitPlaceholder := "$" + strconv.Itoa(position)
	position++
	offsetPlaceholder := "$" + strconv.Itoa(position)

	query := fmt.Sprintf(`
		SELECT `+customerSelectColumns+`
		FROM public.tbl_customers
		%s
		ORDER BY lower(name)
		LIMIT %s OFFSET %s
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search customers: %w", err)
	}
	defer rows.Close()

	customers := make([]CustomerS, 0)
	for rows.Next() {
		var customer CustomerS
		if err := scanCustomer(rows, &customer); err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading customers: %w", err)
	}

	return customers, nil
}
