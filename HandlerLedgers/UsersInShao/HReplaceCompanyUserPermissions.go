package usersinshao

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	userpermissions "github.com/milbertk/shao/dataBasesLedger/userPermissions"
)

// ReplaceCompanyUserPermissionsRequest requires an explicit permissions array.
// Use [] to remove every permission without assigning new ones.
type ReplaceCompanyUserPermissionsRequest struct {
	CompanyUserID string                                    `json:"companyUserId"`
	Permissions   *[]userpermissions.CompanyUserPermissionS `json:"permissions"`
}

type CompanyUserPermissionsHandlerResponse struct {
	Status  string                                       `json:"status"`
	Message string                                       `json:"message"`
	Result  *usersbycompany.CompanyUserPermissionsResult `json:"result,omitempty"`
}

// HReplaceCompanyUserPermissions deletes all existing permissions and inserts
// the complete received permission list.
func HReplaceCompanyUserPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeCompanyUserPermissionsResponse(
			w,
			http.StatusMethodNotAllowed,
			"Use POST",
			nil,
		)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeCompanyUserPermissionsResponse(
			w,
			http.StatusUnauthorized,
			"Missing user claims",
			nil,
		)
		return
	}

	var request ReplaceCompanyUserPermissionsRequest
	if err := decodeSingleJSONObject(w, r, &request); err != nil {
		writeCompanyUserPermissionsResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	request.CompanyUserID = strings.TrimSpace(request.CompanyUserID)
	if request.CompanyUserID == "" {
		writeCompanyUserPermissionsResponse(
			w,
			http.StatusBadRequest,
			"companyUserId is required",
			nil,
		)
		return
	}
	if request.Permissions == nil {
		writeCompanyUserPermissionsResponse(
			w,
			http.StatusBadRequest,
			"permissions is required; use an empty array to remove all permissions",
			nil,
		)
		return
	}

	actorUserID := strings.TrimSpace(claims.UID)
	permissions := *request.Permissions
	for index := range permissions {
		permissions[index].CompanyUserID = request.CompanyUserID
		permissions[index].GrantedByUserID = &actorUserID
	}

	result, err := usersbycompany.BReplaceCompanyUserPermissions(
		request.CompanyUserID,
		permissions,
	)
	if err != nil {
		status := companyUserErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while replacing permissions"
		}

		writeCompanyUserPermissionsResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ReplaceCompanyUserPermissions", handlertracking.Details{"companyUserId": request.CompanyUserID, "permissions": permissions})
	writeCompanyUserPermissionsResponse(
		w,
		http.StatusOK,
		"Company-user permissions replaced successfully.",
		result,
	)
}

func writeCompanyUserPermissionsResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result *usersbycompany.CompanyUserPermissionsResult,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(CompanyUserPermissionsHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}
