package usersinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
)

type myAccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeMyAccessResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(myAccessResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// HSelectMyAccess returns the menu access of the CURRENT user in the selected
// company: { role, fullAccess, permissions: ["NAV_..."] }. The frontend uses
// it to show only the allowed menu options.
//
//	GET /companies/{companyId}/my-access
//
// User, company and role come from the security middleware
// (RequireActiveCompanyMember), never from the request.
func HSelectMyAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeMyAccessResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeMyAccessResponse(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeMyAccessResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	role, _ := security.GetActiveCompanyRole(r)

	result, err := usersbycompany.BSelectMyAccess(companyID, claims.UID, role)
	if err != nil {
		log.Printf("[HSelectMyAccess] Business operation failed: companyId=%s error=%v", companyID, err)
		writeMyAccessResponse(w, http.StatusInternalServerError,
			"An internal error occurred while reading your access", nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectMyAccess", handlertracking.Details{
		"fullAccess":       result.FullAccess,
		"permissionsCount": len(result.Permissions),
	})
	writeMyAccessResponse(w, http.StatusOK, "Access retrieved successfully.", result)
}
