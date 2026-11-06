package categories

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// SelectActiveCategoryNames returns category_id -> category_name for the
// ACTIVE categories of one company. Used by the public store to show
// category chips with real names (never the internal id).
func SelectActiveCategoryNames(companyID string) (map[string]string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	rows, err := db.Query(`
		SELECT category_id::text, category_name
		FROM public.tbl_categories
		WHERE company_id::text = $1
		  AND active = TRUE
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select category names: %w", err)
	}
	defer rows.Close()

	names := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("failed to scan category name: %w", err)
		}
		names[strings.TrimSpace(id)] = strings.TrimSpace(name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading category names: %w", err)
	}
	return names, nil
}
