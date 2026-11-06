package middlewares

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// fullAccessRoles always see every option, regardless of individual permissions.
var fullAccessRoles = map[string]struct{}{
	"OWNER":   {},
	"MANAGER": {},
}

// RequireOptionPermission returns a middleware that allows the request only if
// the current user may access the given option. `optionCode` is the permission
// key stored in the database (a NAV_* code, e.g. "NAV_CUSTOMERS_ACCESS").
//
// IMPORTANT: it must run AFTER security.RequireActiveCompanyMember (or
// RequireActiveCompanyUser), because it reads the validated user, company and
// role from the context those middlewares populate.
//
// optionCode may hold SEVERAL codes separated by commas (2026-10-24): the
// request passes when the user has ANY of them (an action shared by several
// screens, e.g. "NAV_SELF_BOOKING,NAV_APPOINTMENTS_ACCESS").
//
// Rule:
//   - EMPTY optionCode -> no option gate; the request passes as long as the
//     earlier middleware validated the user and company membership. Use this as
//     a placeholder and fill in the NAV_* code later to enable the gate.
//   - OWNER / MANAGER   -> always allowed (full access).
//   - any other role    -> allowed only if the permission exists for that user.
func RequireOptionPermission(
	optionCode string,
) func(http.Handler) http.Handler {
	optionCode = strings.TrimSpace(optionCode)
	var optionCodes []string
	for _, code := range strings.Split(optionCode, ",") {
		if code = strings.TrimSpace(code); code != "" {
			optionCodes = append(optionCodes, code)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No option code configured yet: pass through (membership already
			// validated by the previous middleware).
			if len(optionCodes) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// 1) Validated user (from the JWT claims placed in context).
			claims, ok := security.GetClaims(r)
			if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			userID := strings.TrimSpace(claims.UID)

			// 2) Validated company (from the company middleware).
			companyID, ok := security.GetActiveCompanyID(r)
			companyID = strings.TrimSpace(companyID)
			if !ok || companyID == "" {
				http.Error(w, "Missing validated company context", http.StatusForbidden)
				return
			}

			// 3) OWNER / MANAGER bypass: full access to every option.
			role, _ := security.GetActiveCompanyRole(r)
			role = strings.ToUpper(strings.TrimSpace(role))
			if _, isFullAccess := fullAccessRoles[role]; isFullAccess {
				log.Printf(
					"[RequireOptionPermission] Full-access role granted: option=%s role=%s user=%s",
					optionCode, role, userID,
				)
				next.ServeHTTP(w, r)
				return
			}

			// 4) Resolve the shao company_user_id for this (company, user).
			companyUserID, err := resolveCompanyUserID(companyID, userID)
			if err != nil {
				log.Printf("[RequireOptionPermission] Membership resolution failed: %v", err)
				http.Error(w, "Failed to resolve company membership", http.StatusInternalServerError)
				return
			}
			if companyUserID == "" {
				http.Error(w, "User does not belong to this company", http.StatusForbidden)
				return
			}

			// 5) Validate the option permission(s) using the existing validator:
			//    any of the codes is enough.
			allowed := false
			for _, code := range optionCodes {
				ok, err := usersbycompany.BValidateCompanyUserPermission(companyUserID, code)
				if err != nil {
					log.Printf("[RequireOptionPermission] Permission validation failed: %v", err)
					http.Error(w, "Failed to validate option permission", http.StatusInternalServerError)
					return
				}
				if ok {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Printf(
					"[RequireOptionPermission] Access denied: option=%s user=%s companyUserId=%s",
					optionCode, userID, companyUserID,
				)
				http.Error(w, "You do not have access to this option", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolveCompanyUserID finds the shao company_user_id for a company + user by
// reusing the existing data-layer function. Returns "" when the user has no
// membership in that company.
func resolveCompanyUserID(companyID string, userID string) (string, error) {
	users, err := usersinplatform.SelectCompanyUsersByCompanyID(companyID)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if strings.TrimSpace(user.UserID) == userID {
			return strings.TrimSpace(user.CompanyUserID), nil
		}
	}

	return "", nil
}
