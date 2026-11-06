package usersbycompany

import (
	"errors"
	"fmt"
	"strings"

	userpermissions "github.com/milbertk/shao/dataBasesLedger/userPermissions"
)

// BValidateCompanyUserPermission verifies whether a company-user membership
// has a specific permission.
//
// Returns:
//   - true, nil: permission granted
//   - false, nil: permission not granted
//   - false, error: validation or database failure
func BValidateCompanyUserPermission(
	companyUserID string,
	permissionCode string,
) (bool, error) {
	companyUserID = strings.TrimSpace(companyUserID)
	permissionCode = strings.TrimSpace(permissionCode)

	if companyUserID == "" {
		return false, errors.New(
			"companyUserId is required",
		)
	}

	if permissionCode == "" {
		return false, errors.New(
			"permissionCode is required",
		)
	}

	validator :=
		userpermissions.CompanyUserPermissionValidator{
			CompanyUserID:  companyUserID,
			PermissionCode: permissionCode,
		}

	hasPermission, err := validator.Validate()
	if err != nil {
		return false, fmt.Errorf(
			"failed to validate permission %q for company user %q: %w",
			permissionCode,
			companyUserID,
			err,
		)
	}

	return hasPermission, nil
}
