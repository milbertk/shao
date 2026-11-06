package usersbycompany

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// CompanyUserSelectRequest identifies one membership within one company.
// Both identifiers are required to preserve the company boundary.
type CompanyUserSelectRequest struct {
	CompanyID     string `json:"companyId"`
	CompanyUserID string `json:"companyUserId"`
}

// BSelectCompanyUser validates the required identifiers and returns one
// company-user membership. The data layer applies both values in its WHERE
// clause.
func BSelectCompanyUser(
	request CompanyUserSelectRequest,
) (*usersinplatform.CompanyUserS, error) {
	request.CompanyID = strings.TrimSpace(request.CompanyID)
	request.CompanyUserID = strings.TrimSpace(request.CompanyUserID)

	if err := validateCompanyUserSelectRequest(request); err != nil {
		println("this its the error")
		return nil, err
	}

	user := usersinplatform.CompanyUserS{
		CompanyID:     request.CompanyID,
		CompanyUserID: request.CompanyUserID,
	}

	if err := user.Select(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"company user %q does not exist in company %q",
				request.CompanyUserID,
				request.CompanyID,
			)
		}

		return nil, fmt.Errorf("company-user selection failed: %w", err)
	}

	return &user, nil
}

func validateCompanyUserSelectRequest(request CompanyUserSelectRequest) error {
	if request.CompanyID == "" {
		return errors.New("companyId is required")
	}
	if request.CompanyUserID == "" {
		return errors.New("companyUserId is required")
	}

	return nil
}
