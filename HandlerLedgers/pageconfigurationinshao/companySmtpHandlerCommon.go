package pageconfigurationinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	blcompanysmtp "github.com/milbertk/shao/bussunessLedger/blcompanysmtp"
)

// companySmtpScope validates the logged user and returns the active company.
// It writes the error response itself and returns ok=false on failure.
func companySmtpScope(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writePageConfigurationResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return "", false
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writePageConfigurationResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return "", false
	}
	return companyID, true
}

// decodeCompanySmtp decodes exactly one JSON object (max 16 KB).
func decodeCompanySmtp(r *http.Request) (blcompanysmtp.CompanySmtpInput, error) {
	var input blcompanysmtp.CompanySmtpInput
	if r.Body == nil {
		return input, errors.New("request body is required")
	}

	decoder := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return input, fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return input, errors.New("body must contain a single JSON object")
	}
	return input, nil
}
