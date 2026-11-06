package plans

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CountActiveCompanyUsers counts the active members of a company
// (public.tbl_company_user.active = TRUE).
func CountActiveCompanyUsers(companyID string) (int, error) {
	return countRows(`
		SELECT COUNT(*)
		FROM public.tbl_company_user
		WHERE company_id::text = $1
		  AND active = TRUE
	`, companyID)
}

// CountActiveProducts counts the products of a company that are not
// INACTIVE (deactivated products do not use the plan limit).
func CountActiveProducts(companyID string) (int, error) {
	return countRows(`
		SELECT COUNT(*)
		FROM public.tbl_products
		WHERE company_id::text = $1
		  AND UPPER(COALESCE(status, '')) <> 'INACTIVE'
	`, companyID)
}

func countRows(query, companyID string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var count int
	if err := db.QueryRow(query, strings.TrimSpace(companyID)).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}
	return count, nil
}
