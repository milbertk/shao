package blplan

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
)

// =====================================================================
//  Basic company (2026-10-29).
//
//  The "basic company" is the minimum a company sets up in AlephTelos
//  (Company page) before it can request a plan: its name, logo and header.
//  It can be set up WITHOUT a plan (the Company page save and its two images
//  are not blocked by the read-only mode), and a plan request is refused
//  with code BASIC_COMPANY_REQUIRED until the three are there.
// =====================================================================

// Basic company fields, as sent to the screens in result.missing.
const (
	BasicFieldName   = "name"
	BasicFieldLogo   = "logo"
	BasicFieldHeader = "header"
)

// CodeBasicCompanyRequired is sent in result.code when a plan is requested
// before the basic company is complete.
const CodeBasicCompanyRequired = "BASIC_COMPANY_REQUIRED"

// BasicCompanyError tells which basic company fields are missing.
type BasicCompanyError struct {
	Code    string   `json:"code"`
	Missing []string `json:"missing"`
}

func (e *BasicCompanyError) Error() string {
	return "First create your basic company (name, logo and header) in Company page, then request a plan. Missing: " +
		strings.Join(e.Missing, ", ")
}

// AsBasicCompanyError returns the basic company error inside err, if any.
func AsBasicCompanyError(err error) (*BasicCompanyError, bool) {
	var basicErr *BasicCompanyError
	if errors.As(err, &basicErr) {
		return basicErr, true
	}
	return nil, false
}

func emptyText(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}

// missingBasicCompanyFields lists what the page configuration still needs
// (pure logic, tested). nil config = nothing saved yet.
func missingBasicCompanyFields(config *pageconfiguration.PageConfigurationS) []string {
	if config == nil {
		return []string{BasicFieldName, BasicFieldLogo, BasicFieldHeader}
	}
	missing := []string{}
	if strings.TrimSpace(config.DisplayName) == "" {
		missing = append(missing, BasicFieldName)
	}
	if emptyText(config.LogoURL) {
		missing = append(missing, BasicFieldLogo)
	}
	if emptyText(config.HeaderURL) {
		missing = append(missing, BasicFieldHeader)
	}
	return missing
}

// BBasicCompanyMissing returns the basic company fields the company still
// has to set up in Company page (empty slice = complete).
func BBasicCompanyMissing(companyID string) ([]string, error) {
	config := pageconfiguration.PageConfigurationS{CompanyID: strings.TrimSpace(companyID)}
	if err := config.SelectByCompany(); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return missingBasicCompanyFields(nil), nil
		}
		return nil, fmt.Errorf("failed to read the basic company: %w", err)
	}
	return missingBasicCompanyFields(&config), nil
}

// BCheckBasicCompany returns a *BasicCompanyError when the basic company is
// not complete.
func BCheckBasicCompany(companyID string) error {
	missing, err := BBasicCompanyMissing(companyID)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return &BasicCompanyError{Code: CodeBasicCompanyRequired, Missing: missing}
	}
	return nil
}
