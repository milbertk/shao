package usersbycompany

import (
	"errors"
	"fmt"
	"strings"

	userpermissions "github.com/milbertk/shao/dataBasesLedger/userPermissions"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// MyAccessResult is the menu access of the current user in one company.
// FullAccess is true for OWNER / MANAGER (they see every option).
type MyAccessResult struct {
	Role        string   `json:"role"`
	FullAccess  bool     `json:"fullAccess"`
	Permissions []string `json:"permissions"`
}

// myAccessFullRoles always see every menu option (same rule as the
// RequireOptionPermission middleware).
var myAccessFullRoles = map[string]struct{}{
	"OWNER":   {},
	"MANAGER": {},
}

// BSelectMyAccess returns the role and the NAV_* permission codes of userID in
// companyID. The role comes from the validated session (security middleware),
// never from the request body.
func BSelectMyAccess(companyID, userID, role string) (*MyAccessResult, error) {
	companyID = strings.TrimSpace(companyID)
	userID = strings.TrimSpace(userID)
	role = strings.ToUpper(strings.TrimSpace(role))

	if companyID == "" || userID == "" {
		return nil, errors.New("companyId and user are required")
	}

	result := &MyAccessResult{Role: role, Permissions: []string{}}
	if _, ok := myAccessFullRoles[role]; ok {
		result.FullAccess = true
		return result, nil
	}

	users, err := usersinplatform.SelectCompanyUsersByCompanyID(companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to read company users: %w", err)
	}

	companyUserID := ""
	for _, user := range users {
		if strings.TrimSpace(user.UserID) == userID {
			companyUserID = strings.TrimSpace(user.CompanyUserID)
			break
		}
	}
	if companyUserID == "" {
		// Not a shao member of this company: no options.
		return result, nil
	}

	permissions, err := userpermissions.SelectPermissionsByCompanyUserID(companyUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to read permissions: %w", err)
	}
	for _, permission := range permissions {
		code := strings.TrimSpace(permission.PermissionCode)
		if code != "" {
			result.Permissions = append(result.Permissions, code)
		}
	}
	return result, nil
}
