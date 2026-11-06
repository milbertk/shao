package usersbycompany

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	userpermissions "github.com/milbertk/shao/dataBasesLedger/userPermissions"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// CompanyUserPermissionsResult contains the final permission list after the
// complete replacement operation.
type CompanyUserPermissionsResult struct {
	CompanyUserID string                                   `json:"companyUserId"`
	Permissions   []userpermissions.CompanyUserPermissionS `json:"permissions"`
}

// BReplaceCompanyUserPermissions deletes every existing permission for the
// company user, then inserts the complete received list.
func BReplaceCompanyUserPermissions(
	companyUserID string,
	permissions []userpermissions.CompanyUserPermissionS,
) (*CompanyUserPermissionsResult, error) {
	companyUserID = strings.TrimSpace(companyUserID)
	if companyUserID == "" {
		return nil, errors.New("companyUserId is required")
	}

	companyUser := usersinplatform.CompanyUserS{
		CompanyUserID: companyUserID,
	}
	if err := companyUser.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("company user %q does not exist", companyUserID)
		}
		return nil, fmt.Errorf("failed to validate company user: %w", err)
	}

	normalized, err := normalizePermissionReplacement(companyUserID, permissions)
	if err != nil {
		return nil, err
	}

	existing, err := userpermissions.SelectPermissionsByCompanyUserID(companyUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing permissions: %w", err)
	}

	// The replacement rule always removes every previous permission first.
	for _, permission := range existing {
		if err := permission.Delete(); err != nil {
			return nil, fmt.Errorf(
				"failed to delete permission %q: %w",
				permission.PermissionCode,
				err,
			)
		}
	}

	for _, permission := range normalized {
		if err := permission.Insert(); err != nil {
			return nil, fmt.Errorf(
				"failed to insert permission %q: %w",
				permission.PermissionCode,
				err,
			)
		}
	}

	saved, err := userpermissions.SelectPermissionsByCompanyUserID(companyUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to read saved permissions: %w", err)
	}

	return &CompanyUserPermissionsResult{
		CompanyUserID: companyUserID,
		Permissions:   saved,
	}, nil
}

func normalizePermissionReplacement(
	companyUserID string,
	permissions []userpermissions.CompanyUserPermissionS,
) ([]userpermissions.CompanyUserPermissionS, error) {
	normalized := make([]userpermissions.CompanyUserPermissionS, 0, len(permissions))
	seen := make(map[string]struct{}, len(permissions))

	for _, permission := range permissions {
		permission.PermissionCode = strings.TrimSpace(permission.PermissionCode)
		if permission.PermissionCode == "" {
			return nil, errors.New("permissionCode is required")
		}
		if _, exists := seen[permission.PermissionCode]; exists {
			return nil, fmt.Errorf(
				"permissionCode %q has been provided more than once",
				permission.PermissionCode,
			)
		}

		permission.CompanyUserID = companyUserID
		seen[permission.PermissionCode] = struct{}{}
		normalized = append(normalized, permission)
	}

	return normalized, nil
}
