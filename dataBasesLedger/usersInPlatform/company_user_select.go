package usersinplatform

import (
	"fmt"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Select loads one company-user membership using CompanyUserID.
func (a *CompanyUserS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
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
		WHERE company_user_id = $1
	`

	err = db.QueryRow(query, a.CompanyUserID).Scan(
		&a.CompanyUserID,
		&a.CompanyID,
		&a.UserID,
		&a.Email,
		&a.Role,
		&a.Active,
		&a.IsProtected,
		&a.JoinedAt,
		&a.CreatedByUserID,
		&a.UpdatedByUserID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to select company user: %w", err)
	}

	return nil
}

// SelectCompanyUsersByCompanyID returns every user assigned to one company.
func SelectCompanyUsersByCompanyID(companyID string) ([]CompanyUserS, error) {
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
		ORDER BY email
	`

	rows, err := db.Query(query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select company users: %w", err)
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
			return nil, fmt.Errorf("failed to scan company user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading company users: %w", err)
	}

	return users, nil
}
