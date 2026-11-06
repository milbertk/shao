package userpermissions

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyUserPermissionS represents one row from
// public.tbl_company_user_permission.
type CompanyUserPermissionS struct {
	CompanyUserID   string    `json:"companyUserId"`
	PermissionCode  string    `json:"permissionCode"`
	GrantedByUserID *string   `json:"grantedByUserId,omitempty"`
	GrantedAt       time.Time `json:"grantedAt"`
}

// Insert assigns one permission to a company-user membership.
func (a *CompanyUserPermissionS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_company_user_permission (
			company_user_id,
			permission_code,
			granted_by_user_id
		)
		VALUES ($1, $2, $3)
		RETURNING granted_at
	`

	err = db.QueryRow(
		query,
		a.CompanyUserID,
		a.PermissionCode,
		a.GrantedByUserID,
	).Scan(&a.GrantedAt)
	if err != nil {
		return fmt.Errorf("failed to insert company-user permission: %w", err)
	}

	return nil
}
