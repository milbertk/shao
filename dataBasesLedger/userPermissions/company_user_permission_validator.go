package userpermissions

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyUserPermissionValidator validates whether a company-user
// membership has a specific permission.
type CompanyUserPermissionValidator struct {
	CompanyUserID  string
	PermissionCode string
}

// Validate returns:
//   - true when the company user has the permission
//   - false when the company user does not have the permission
//   - an error when validation or the database query fails
func (v *CompanyUserPermissionValidator) Validate() (bool, error) {
	companyUserID := strings.TrimSpace(v.CompanyUserID)
	permissionCode := strings.TrimSpace(v.PermissionCode)

	if companyUserID == "" {
		return false, fmt.Errorf(
			"company user ID is required",
		)
	}

	if permissionCode == "" {
		return false, fmt.Errorf(
			"permission code is required",
		)
	}

	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return false, fmt.Errorf(
			"database connection error: %w",
			err,
		)
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM public.tbl_company_user_permission
			WHERE company_user_id = $1
			  AND permission_code = $2
		)
	`

	var hasPermission bool

	err = db.QueryRow(
		query,
		companyUserID,
		permissionCode,
	).Scan(&hasPermission)

	if err != nil {
		return false, fmt.Errorf(
			"failed to validate company-user permission: %w",
			err,
		)
	}

	return hasPermission, nil
}
