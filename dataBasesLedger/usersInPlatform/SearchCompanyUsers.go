package usersinplatform

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyUserSearchFilter contains the criteria supported by
// SearchCompanyUsers. CompanyID is mandatory and is always matched exactly.
//
// Search is optional and is matched partially, without case sensitivity,
// against user_id and email.
//
// Active behavior:
//   - nil: active users only (default)
//   - pointer to true: active users only
//   - pointer to false: inactive users only
type CompanyUserSearchFilter struct {
	CompanyID string
	Search    string
	Active    *bool
}

// SearchCompanyUsers returns company-user memberships using criteria intended
// for a search screen. It never returns memberships from another company.
func SearchCompanyUsers(filter CompanyUserSearchFilter) ([]CompanyUserS, error) {
	companyID := strings.TrimSpace(filter.CompanyID)
	search := strings.TrimSpace(filter.Search)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	active := true
	if filter.Active != nil {
		active = *filter.Active
	}

	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			company_user_id,
			company_id,
			user_id,
			email,
			role,
			active,
			is_protected,
			joined_at,
			created_by_user_id,
			updated_by_user_id,
			created_at,
			updated_at
		FROM public.tbl_company_user
		WHERE company_id = $1
		  AND active = $2
		  AND (
			$3 = ''
			OR user_id::text ILIKE '%' || $3 || '%'
			OR email ILIKE '%' || $3 || '%'
		  )
		ORDER BY email, user_id
	`

	rows, err := db.Query(query, companyID, active, search)
	if err != nil {
		return nil, fmt.Errorf("failed to search company users: %w", err)
	}
	defer rows.Close()

	users := make([]CompanyUserS, 0)
	for rows.Next() {
		var user CompanyUserS
		if err := rows.Scan(
			&user.CompanyUserID,
			&user.CompanyID,
			&user.UserID,
			&user.Email,
			&user.Role,
			&user.Active,
			&user.IsProtected,
			&user.JoinedAt,
			&user.CreatedByUserID,
			&user.UpdatedByUserID,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan company user search result: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading company user search results: %w", err)
	}

	return users, nil
}
