package userpermissions

import (
	"fmt"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Select loads one permission using CompanyUserID and PermissionCode.
func (a *CompanyUserPermissionS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			company_user_id,
			permission_code,
			granted_by_user_id,
			granted_at
		FROM public.tbl_company_user_permission
		WHERE company_user_id = $1
		  AND permission_code = $2
	`

	err = db.QueryRow(query, a.CompanyUserID, a.PermissionCode).Scan(
		&a.CompanyUserID,
		&a.PermissionCode,
		&a.GrantedByUserID,
		&a.GrantedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to select company-user permission: %w", err)
	}

	return nil
}

// SelectPermissionsByCompanyUserID returns all permissions assigned to one
// company-user membership.
func SelectPermissionsByCompanyUserID(companyUserID string) ([]CompanyUserPermissionS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			company_user_id,
			permission_code,
			granted_by_user_id,
			granted_at
		FROM public.tbl_company_user_permission
		WHERE company_user_id = $1
		ORDER BY permission_code
	`

	rows, err := db.Query(query, companyUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to select company-user permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]CompanyUserPermissionS, 0)
	for rows.Next() {
		var permission CompanyUserPermissionS
		if err := rows.Scan(
			&permission.CompanyUserID,
			&permission.PermissionCode,
			&permission.GrantedByUserID,
			&permission.GrantedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan company-user permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading company-user permissions: %w", err)
	}

	return permissions, nil
}
