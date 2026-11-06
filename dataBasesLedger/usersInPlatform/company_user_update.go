package usersinplatform

import (
	"database/sql"
	"fmt"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies the editable fields of one company-user membership.
func (a *CompanyUserS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		UPDATE public.tbl_company_user
		SET
			email = $1,
			role = $2,
			active = $3,
			is_protected = $4,
			joined_at = $5,
			updated_by_user_id = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE company_user_id = $7
		RETURNING updated_at
	`

	err = db.QueryRow(
		query,
		a.Email,
		a.Role,
		a.Active,
		a.IsProtected,
		a.JoinedAt,
		a.UpdatedByUserID,
		a.CompanyUserID,
	).Scan(&a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("company user %q does not exist", a.CompanyUserID)
		}
		return fmt.Errorf("failed to update company user: %w", err)
	}

	return nil
}

// Delete removes one company-user membership. Its permissions are removed by
// the table's ON DELETE CASCADE foreign key.
func (a *CompanyUserS) Delete() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_company_user
		WHERE company_user_id = $1
	`, a.CompanyUserID)
	if err != nil {
		return fmt.Errorf("failed to delete company user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify deleted company user: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
