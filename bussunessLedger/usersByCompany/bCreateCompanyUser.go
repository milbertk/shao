package usersbycompany

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/blplan"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// CompanyUserResult contains the create/update decision and the saved user.
type CompanyUserResult struct {
	Action string                       `json:"action"`
	User   usersinplatform.CompanyUserS `json:"user"`
}

// BCreateOrUpdateCompanyUser creates a membership when CompanyUserID is empty.
// When CompanyUserID is populated, it updates only Active and the update audit
// user. All other stored fields remain unchanged.
func BCreateOrUpdateCompanyUser(
	user usersinplatform.CompanyUserS,
) (*CompanyUserResult, error) {
	normalizeCompanyUser(&user)

	if user.CompanyUserID == "" {
		if err := validateNewCompanyUser(user); err != nil {
			return nil, err
		}
		if err := validateMembershipDoesNotExist(user.CompanyID, user.UserID); err != nil {
			return nil, err
		}
		// Plan limit of active users (2026-10-22): *blplan.PlanLimitError.
		if err := blplan.BCheckCanAddUser(user.CompanyID); err != nil {
			return nil, err
		}

		// New memberships are active by default.
		user.Active = true
		if err := user.Insert(); err != nil {
			return nil, fmt.Errorf("company-user creation failed: %w", err)
		}

		return &CompanyUserResult{
			Action: "create",
			User:   user,
		}, nil
	}

	existing := usersinplatform.CompanyUserS{
		CompanyUserID: user.CompanyUserID,
	}
	if err := existing.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"company user %q does not exist",
				user.CompanyUserID,
			)
		}
		return nil, fmt.Errorf("error validating companyUserId: %w", err)
	}

	// Re-activating a user also uses one place of the plan (2026-10-22).
	if user.Active && !existing.Active {
		if err := blplan.BCheckCanAddUser(existing.CompanyID); err != nil {
			return nil, err
		}
	}

	// Active is intentionally the only business field changed during update.
	existing.Active = user.Active
	existing.UpdatedByUserID = user.UpdatedByUserID
	if err := existing.Update(); err != nil {
		return nil, fmt.Errorf("company-user active-status update failed: %w", err)
	}

	return &CompanyUserResult{
		Action: "update",
		User:   existing,
	}, nil
}

func normalizeCompanyUser(user *usersinplatform.CompanyUserS) {
	user.CompanyUserID = strings.TrimSpace(user.CompanyUserID)
	user.CompanyID = strings.TrimSpace(user.CompanyID)
	user.UserID = strings.TrimSpace(user.UserID)
	user.Email = strings.TrimSpace(user.Email)
	user.Role = strings.TrimSpace(user.Role)

	if user.Role == "" {
		user.Role = "operator"
	}
}

func validateNewCompanyUser(user usersinplatform.CompanyUserS) error {
	if user.CompanyID == "" {
		return errors.New("companyId is required")
	}
	if user.UserID == "" {
		return errors.New("userId is required")
	}
	//if user.Email == "" {
	//	return errors.New("email is required")
	//}
	//if !strings.Contains(user.Email, "@") {
	//	return errors.New("email must contain the @ symbol")
	//}
	if user.Role == "" {
		return errors.New("role is required")
	}

	return nil
}

func validateMembershipDoesNotExist(companyID, userID string) error {
	users, err := usersinplatform.SelectCompanyUsersByCompanyID(companyID)
	if err != nil {
		return fmt.Errorf("failed to validate existing company users: %w", err)
	}

	for _, existing := range users {
		if strings.TrimSpace(existing.UserID) == userID {
			return fmt.Errorf(
				"user %q already belongs to company %q",
				userID,
				companyID,
			)
		}
	}

	return nil
}
