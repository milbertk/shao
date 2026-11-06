package userpermissions

import (
	"database/sql"
	"fmt"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update changes a permission code and its grant audit information.
// previousPermissionCode identifies the existing composite primary-key value.
func (a *CompanyUserPermissionS) Update(previousPermissionCode string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		UPDATE public.tbl_company_user_permission
		SET
			permission_code = $1,
			granted_by_user_id = $2,
			granted_at = CURRENT_TIMESTAMP
		WHERE company_user_id = $3
		  AND permission_code = $4
		RETURNING granted_at
	`

	err = db.QueryRow(
		query,
		a.PermissionCode,
		a.GrantedByUserID,
		a.CompanyUserID,
		previousPermissionCode,
	).Scan(&a.GrantedAt)
	if err != nil {
		return fmt.Errorf("failed to update company-user permission: %w", err)
	}

	return nil
}

// Delete removes one permission from one company-user membership.
func (a *CompanyUserPermissionS) Delete() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_company_user_permission
		WHERE company_user_id = $1
		  AND permission_code = $2
	`, a.CompanyUserID, a.PermissionCode)
	if err != nil {
		return fmt.Errorf("failed to delete company-user permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify deleted company-user permission: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
