package usersinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

type CompanyUserSelectHandlerResponse struct {
	Status  string                        `json:"status"`
	Message string                        `json:"message"`
	Result  *usersinplatform.CompanyUserS `json:"result,omitempty"`
}

// HSelectCompanyUser returns one membership from the company already
// validated by security.RequireActiveCompanyUser.
func HSelectCompanyUser(w http.ResponseWriter, r *http.Request) {
	// This is the first handler checkpoint. If the middleware message appears
	// but this message does not, the middleware has not called the next handler.
	log.Printf(
		"[HSelectCompanyUser] Handler reached: method=%s path=%s",
		r.Method,
		r.URL.Path,
	)

	if r.Method != http.MethodGet {
		log.Printf(
			"[HSelectCompanyUser] Invalid method: %s",
			r.Method,
		)

		w.Header().Set("Allow", http.MethodGet)
		writeCompanyUserSelectResponse(
			w,
			http.StatusMethodNotAllowed,
			"Use GET",
			nil,
		)
		return
	}

	activeCompanyID, ok := security.GetActiveCompanyID(r)
	activeCompanyID = strings.TrimSpace(activeCompanyID)
	if !ok || activeCompanyID == "" {
		log.Println("[HSelectCompanyUser] Missing company context")

		writeCompanyUserSelectResponse(
			w,
			http.StatusForbidden,
			"Missing validated company context",
			nil,
		)
		return
	}

	log.Printf(
		"[HSelectCompanyUser] Company context has been validated: companyId=%s",
		activeCompanyID,
	)

	routeVariables := mux.Vars(r)
	log.Printf(
		"[HSelectCompanyUser] Route variables received: companyId=%s companyUserId=%s",
		routeVariables["companyId"],
		routeVariables["companyUserId"],
	)

	companyUserID := strings.TrimSpace(routeVariables["companyUserId"])
	if companyUserID == "" {
		log.Println("[HSelectCompanyUser] Missing companyUserId route parameter")

		writeCompanyUserSelectResponse(
			w,
			http.StatusBadRequest,
			"companyUserId is required",
			nil,
		)
		return
	}

	log.Printf(
		"[HSelectCompanyUser] Calling BSelectCompanyUser: companyId=%s companyUserId=%s",
		activeCompanyID,
		companyUserID,
	)

	user, err := usersbycompany.BSelectCompanyUser(
		usersbycompany.CompanyUserSelectRequest{
			CompanyID:     activeCompanyID,
			CompanyUserID: companyUserID,
		},
	)
	if err != nil {
		status := companyUserSelectErrorStatus(err)

		log.Printf(
			"[HSelectCompanyUser] Business operation has failed: status=%d error=%v",
			status,
			err,
		)

		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the company user"
		}

		writeCompanyUserSelectResponse(w, status, message, nil)
		return
	}

	if user == nil {
		log.Println("[HSelectCompanyUser] Business operation returned no user")

		writeCompanyUserSelectResponse(
			w,
			http.StatusNotFound,
			"Company user does not exist",
			nil,
		)
		return
	}

	log.Printf(
		"[HSelectCompanyUser] Company user has been found: companyUserId=%s",
		user.CompanyUserID,
	)

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCompanyUser", handlertracking.Details{"companyUserId": companyUserID})
	writeCompanyUserSelectResponse(
		w,
		http.StatusOK,
		"Company user retrieved successfully.",
		user,
	)

	log.Println("[HSelectCompanyUser] Successful response has been sent")
}

func companyUserSelectErrorStatus(err error) int {
	message := strings.ToLower(err.Error())

	switch {
	case strings.Contains(message, "does not exist"),
		strings.Contains(message, "no rows"):
		return http.StatusNotFound
	case strings.Contains(message, "is required"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeCompanyUserSelectResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result *usersinplatform.CompanyUserS,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(CompanyUserSelectHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	}); err != nil {
		log.Printf(
			"[HSelectCompanyUser] Error encoding response: %v",
			err,
		)
	}
}
