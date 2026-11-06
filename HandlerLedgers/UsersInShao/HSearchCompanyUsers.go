package usersinshao

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

type CompanyUserSearchHandlerResponse struct {
	Status  string                         `json:"status"`
	Message string                         `json:"message"`
	Result  []usersinplatform.CompanyUserS `json:"result"`
}

// HSearchCompanyUsers searches memberships from the company already validated
// by security.RequireActiveCompanyUser.
//
// Query parameters:
//   - search: optional partial user ID or email
//   - active: required; true or false
func HSearchCompanyUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeCompanyUserSearchResponse(
			w,
			http.StatusMethodNotAllowed,
			"Use GET",
			nil,
		)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeCompanyUserSearchResponse(
			w,
			http.StatusForbidden,
			"Missing validated company context",
			nil,
		)
		return
	}

	active, err := parseRequiredActive(r.URL.Query().Get("active"))
	if err != nil {
		writeCompanyUserSearchResponse(
			w,
			http.StatusBadRequest,
			err.Error(),
			nil,
		)
		return
	}

	users, err := usersbycompany.BSearchCompanyUsers(
		usersbycompany.CompanyUserSearchRequest{
			CompanyID: companyID,
			Search:    r.URL.Query().Get("search"),
			Active:    &active,
		},
	)
	if err != nil {
		status := companyUserSearchErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching company users"
		}

		writeCompanyUserSearchResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchCompanyUsers", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(users)})
	writeCompanyUserSearchResponse(
		w,
		http.StatusOK,
		"Company users retrieved successfully.",
		users,
	)
}

func parseRequiredActive(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "":
		return false, &companyUserSearchParameterError{
			message: "active is required",
		}
	default:
		return false, &companyUserSearchParameterError{
			message: "active must be true or false",
		}
	}
}

type companyUserSearchParameterError struct {
	message string
}

func (e *companyUserSearchParameterError) Error() string {
	return e.message
}

func companyUserSearchErrorStatus(err error) int {
	if strings.Contains(strings.ToLower(err.Error()), "is required") {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

func writeCompanyUserSearchResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result []usersinplatform.CompanyUserS,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(CompanyUserSearchHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}
