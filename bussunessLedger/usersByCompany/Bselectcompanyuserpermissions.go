package usersbycompany

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	userpermissions "github.com/milbertk/shao/dataBasesLedger/userPermissions"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// BSelectCompanyUserPermissions validates that the company-user membership
// exists and returns the permissions currently assigned to it.
//
// It follows the same shape as BReplaceCompanyUserPermissions:
//  1. requires a non-empty companyUserID,
//  2. confirms the membership exists (distinguishing "not found" from a real
//     database error),
//  3. reads the current permissions via the database layer,
//  4. returns them in the shared CompanyUserPermissionsResult struct.
//
// It reuses CompanyUserPermissionsResult, which is already declared in this
// package (bReplaceCompanyUserPermissions.go), so it is NOT redefined here.
func BSelectCompanyUserPermissions(
	companyUserID string,
) (*CompanyUserPermissionsResult, error) {
	companyUserID = strings.TrimSpace(companyUserID)
	if companyUserID == "" {
		return nil, errors.New("companyUserId is required")
	}

	// Validate that the company-user membership exists.
	companyUser := usersinplatform.CompanyUserS{
		CompanyUserID: companyUserID,
	}
	if err := companyUser.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("company user %q does not exist", companyUserID)
		}
		return nil, fmt.Errorf("failed to validate company user: %w", err)
	}

	// Read the permissions currently assigned to this membership.
	permissions, err := userpermissions.SelectPermissionsByCompanyUserID(companyUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to read permissions: %w", err)
	}

	return &CompanyUserPermissionsResult{
		CompanyUserID: companyUserID,
		Permissions:   permissions,
	}, nil
}
