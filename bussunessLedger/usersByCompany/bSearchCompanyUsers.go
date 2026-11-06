package usersbycompany

import (
	"errors"
	"fmt"
	"strings"

	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

// CompanyUserSearchRequest contains the criteria accepted by the company-user
// search business layer.
//
// Search may be empty; an empty value returns every membership matching the
// company and Active filters. Active is a pointer so false can be distinguished
// from an omitted JSON field.
type CompanyUserSearchRequest struct {
	CompanyID string `json:"companyId"`
	Search    string `json:"search"`
	Active    *bool  `json:"active"`
}

// BSearchCompanyUsers validates the required search fields and returns the
// matching company-user memberships.
func BSearchCompanyUsers(
	request CompanyUserSearchRequest,
) ([]usersinplatform.CompanyUserS, error) {
	request.CompanyID = strings.TrimSpace(request.CompanyID)
	request.Search = strings.TrimSpace(request.Search)

	if err := validateCompanyUserSearchRequest(request); err != nil {
		return nil, err
	}

	users, err := usersinplatform.SearchCompanyUsers(
		usersinplatform.CompanyUserSearchFilter{
			CompanyID: request.CompanyID,
			Search:    request.Search,
			Active:    request.Active,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("company-user search failed: %w", err)
	}

	return users, nil
}

func validateCompanyUserSearchRequest(request CompanyUserSearchRequest) error {
	if request.CompanyID == "" {
		return errors.New("companyId is required")
	}
	if request.Active == nil {
		return errors.New("active is required")
	}

	// Search is intentionally optional. An empty value means all company users
	// with the requested active status.
	return nil
}
