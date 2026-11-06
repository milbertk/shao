package usersinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
)

// HSelectCompanyUserPermissions returns the permissions currently assigned to
// one company-user membership.
//
// It mirrors HSelectCompanyUser: GET only, company context validated by
// security.RequireActiveCompanyUser, companyUserId taken from the route.
//
// It reuses the helpers already declared in this package by the replace
// handler (writeCompanyUserPermissionsResponse, companyUserErrorStatus,
// CompanyUserPermissionsHandlerResponse), so none of those are redefined here.
func HSelectCompanyUserPermissions(w http.ResponseWriter, r *http.Request) {
	log.Printf(
		"[HSelectCompanyUserPermissions] Handler reached: method=%s path=%s",
		r.Method,
		r.URL.Path,
	)

	if r.Method != http.MethodGet {
		log.Printf(
			"[HSelectCompanyUserPermissions] Invalid method: %s",
			r.Method,
		)

		w.Header().Set("Allow", http.MethodGet)
		writeCompanyUserPermissionsResponse(
			w,
			http.StatusMethodNotAllowed,
			"Use GET",
			nil,
		)
		return
	}

	// The company must have been validated by the middleware chain.
	activeCompanyID, ok := security.GetActiveCompanyID(r)
	activeCompanyID = strings.TrimSpace(activeCompanyID)
	if !ok || activeCompanyID == "" {
		log.Println("[HSelectCompanyUserPermissions] Missing company context")

		writeCompanyUserPermissionsResponse(
			w,
			http.StatusForbidden,
			"Missing validated company context",
			nil,
		)
		return
	}

	log.Printf(
		"[HSelectCompanyUserPermissions] Company context has been validated: companyId=%s",
		activeCompanyID,
	)

	routeVariables := mux.Vars(r)
	log.Printf(
		"[HSelectCompanyUserPermissions] Route variables received: companyId=%s companyUserId=%s",
		routeVariables["companyId"],
		routeVariables["companyUserId"],
	)

	companyUserID := strings.TrimSpace(routeVariables["companyUserId"])
	if companyUserID == "" {
		log.Println("[HSelectCompanyUserPermissions] Missing companyUserId route parameter")

		writeCompanyUserPermissionsResponse(
			w,
			http.StatusBadRequest,
			"companyUserId is required",
			nil,
		)
		return
	}

	log.Printf(
		"[HSelectCompanyUserPermissions] Calling BSelectCompanyUserPermissions: companyId=%s companyUserId=%s",
		activeCompanyID,
		companyUserID,
	)

	result, err := usersbycompany.BSelectCompanyUserPermissions(companyUserID)
	if err != nil {
		status := companyUserErrorStatus(err)

		log.Printf(
			"[HSelectCompanyUserPermissions] Business operation has failed: status=%d error=%v",
			status,
			err,
		)

		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the permissions"
		}

		writeCompanyUserPermissionsResponse(w, status, message, nil)
		return
	}

	log.Printf(
		"[HSelectCompanyUserPermissions] Permissions retrieved: companyUserId=%s count=%d",
		companyUserID,
		len(result.Permissions),
	)

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCompanyUserPermissions", handlertracking.Details{"companyUserId": companyUserID})
	writeCompanyUserPermissionsResponse(
		w,
		http.StatusOK,
		"Company-user permissions retrieved successfully.",
		result,
	)

	log.Println("[HSelectCompanyUserPermissions] Successful response has been sent")
}
