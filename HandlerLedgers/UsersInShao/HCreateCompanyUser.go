package usersinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	usersbycompany "github.com/milbertk/shao/bussunessLedger/usersByCompany"
	usersinplatform "github.com/milbertk/shao/dataBasesLedger/usersInPlatform"
)

const maxCompanyUserRequestBytes int64 = 1 << 20 // 1 MiB

const (
	companyRoleOwner   = "OWNER"
	companyRoleManager = "MANAGER"
)

type CompanyUserHandlerResponse struct {
	Status  string                            `json:"status"`
	Message string                            `json:"message"`
	Result  *usersbycompany.CompanyUserResult `json:"result,omitempty"`
}

// HCreateOrUpdateCompanyUser creates a company user or updates only Active.
//
// This handler must be protected with security.RequireActiveCompanyUser.
// That middleware authenticates the user, validates that the user is active,
// validates company access, and stores the verified company ID and company
// role in the request context.
func HCreateOrUpdateCompanyUser(w http.ResponseWriter, r *http.Request) {
	// This is the first handler checkpoint. If this message does not appear,
	// the router or RequireActiveCompanyUser has stopped the request first.
	log.Printf(
		"[HCreateOrUpdateCompanyUser] Handler reached: method=%s path=%s",
		r.Method,
		r.URL.Path,
	)

	if r.Method != http.MethodPost {
		log.Printf(
			"[HCreateOrUpdateCompanyUser] Invalid method: %s",
			r.Method,
		)

		w.Header().Set("Allow", http.MethodPost)
		writeCompanyUserResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		log.Println("[HCreateOrUpdateCompanyUser] Missing user claims")
		writeCompanyUserResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	log.Println("[HCreateOrUpdateCompanyUser] Claims have been validated")

	activeCompanyID, ok := security.GetActiveCompanyID(r)
	activeCompanyID = strings.TrimSpace(activeCompanyID)
	if !ok || activeCompanyID == "" {
		log.Println("[HCreateOrUpdateCompanyUser] Missing company context")
		writeCompanyUserResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	log.Printf(
		"[HCreateOrUpdateCompanyUser] Company context has been validated: companyId=%s",
		activeCompanyID,
	)

	activeCompanyRole, ok := security.GetActiveCompanyRole(r)
	activeCompanyRole = strings.ToUpper(strings.TrimSpace(activeCompanyRole))
	if !ok || activeCompanyRole == "" {
		log.Println("[HCreateOrUpdateCompanyUser] Missing company role")
		writeCompanyUserResponse(w, http.StatusForbidden, "Missing validated company role", nil)
		return
	}

	log.Printf(
		"[HCreateOrUpdateCompanyUser] Role has been validated: role=%s",
		activeCompanyRole,
	)

	if activeCompanyRole != companyRoleOwner && activeCompanyRole != companyRoleManager {
		log.Printf(
			"[HCreateOrUpdateCompanyUser] Access denied for role=%s",
			activeCompanyRole,
		)

		writeCompanyUserResponse(
			w,
			http.StatusForbidden,
			"Only a company owner or manager can create or update company users",
			nil,
		)
		return
	}

	var user usersinplatform.CompanyUserS
	if err := decodeSingleJSONObject(w, r, &user); err != nil {
		log.Printf(
			"[HCreateOrUpdateCompanyUser] Invalid request body: %v",
			err,
		)

		writeCompanyUserResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HCreateOrUpdateCompanyUser] Body has been decoded: companyUserId=%s userId=%s active=%v",
		user.CompanyUserID,
		user.UserID,
		user.Active,
	)

	// The validated company context is authoritative. A caller cannot perform
	// an operation for a different company by changing the JSON body.
	requestCompanyID := strings.TrimSpace(user.CompanyID)
	if requestCompanyID != "" && requestCompanyID != activeCompanyID {
		log.Printf(
			"[HCreateOrUpdateCompanyUser] Company mismatch: body=%s context=%s",
			requestCompanyID,
			activeCompanyID,
		)

		writeCompanyUserResponse(
			w,
			http.StatusForbidden,
			"The request company does not match the validated active company",
			nil,
		)
		return
	}
	user.CompanyID = activeCompanyID

	// The actor is always obtained from the validated JWT claims. It is never
	// accepted from the frontend.
	actorUserID := strings.TrimSpace(claims.UID)
	user.UpdatedByUserID = &actorUserID

	if strings.TrimSpace(user.CompanyUserID) == "" {
		user.CreatedByUserID = &actorUserID

		// Identity is resolved server-side, never trusted from the body:
		//   - user_id: the validated caller UID.
		//   - email:   the official e-mail the middleware read from
		//              public.usfirebasedata and stored in the context.
		user.UserID = actorUserID

		contextEmail, _ := security.GetActiveCompanyEmail(r)
		contextEmail = strings.TrimSpace(contextEmail)
		if contextEmail == "" {
			// Fallback to the validated token e-mail if usfirebasedata had none.
			contextEmail = strings.TrimSpace(claims.Email)
		}
		user.Email = contextEmail
	}

	log.Println("[HCreateOrUpdateCompanyUser] Calling BCreateOrUpdateCompanyUser")

	result, err := usersbycompany.BCreateOrUpdateCompanyUser(user)
	if err != nil {
		log.Printf(
			"[HCreateOrUpdateCompanyUser] Business operation has failed: %v",
			err,
		)

		// Plan limit (2026-10-22): 403 with result.code = PLAN_LIMIT_USERS.
		if planinshao.WritePlanLimit(w, err) {
			handlertracking.Record(r, "CreateCompanyUserPlanLimit", handlertracking.Details{"error": err.Error()})
			return
		}

		status := companyUserErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the company user"
		}

		writeCompanyUserResponse(w, status, message, nil)
		return
	}
	if result == nil {
		log.Println("[HCreateOrUpdateCompanyUser] Business operation has returned nil")

		writeCompanyUserResponse(
			w,
			http.StatusInternalServerError,
			"The company user operation returned no result",
			nil,
		)
		return
	}

	log.Printf(
		"[HCreateOrUpdateCompanyUser] Operation has succeeded: action=%s",
		result.Action,
	)

	insertCompanyUserTracking(
		r,
		actorUserID,
		claims.Email,
		activeCompanyID,
		activeCompanyRole,
		result,
	)

	status := http.StatusCreated
	message := "Company user created successfully."
	if result.Action == "update" {
		status = http.StatusOK
		message = "Company user active status updated successfully."
	}

	log.Printf(
		"[HCreateOrUpdateCompanyUser] Sending response: status=%d action=%s",
		status,
		result.Action,
	)

	writeCompanyUserResponse(w, status, message, result)
}

// insertCompanyUserTracking records the successful operation. Tracking is
// best-effort: a tracking failure is logged but does not undo the completed
// company-user transaction.
func insertCompanyUserTracking(
	r *http.Request,
	actorUserID string,
	actorEmail string,
	activeCompanyID string,
	activeCompanyRole string,
	result *usersbycompany.CompanyUserResult,
) {
	actionName := "CreateCompanyUser"
	if strings.EqualFold(strings.TrimSpace(result.Action), "update") {
		actionName = "UpdateCompanyUser"
	}

	trackingPayload := struct {
		CompanyID        string                            `json:"companyId"`
		ActorCompanyRole string                            `json:"actorCompanyRole"`
		Result           *usersbycompany.CompanyUserResult `json:"result"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		Result:           result,
	}

	jsonTrackingPayload, err := json.Marshal(trackingPayload)
	if err != nil {
		log.Printf("[HCreateOrUpdateCompanyUser] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readCompanyUserClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         actionName,
		Jsonstring:     string(jsonTrackingPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HCreateOrUpdateCompanyUser] Error inserting tracking: %v", err)
	}
}

func readCompanyUserClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	return r.RemoteAddr
}

func decodeSingleJSONObject(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxCompanyUserRequestBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("the request body must contain only one JSON object")
	}

	return nil
}

func companyUserErrorStatus(err error) int {
	message := strings.ToLower(err.Error())

	switch {
	case strings.Contains(message, "does not exist"),
		strings.Contains(message, "no rows"):
		return http.StatusNotFound

	case strings.Contains(message, "already belongs"),
		strings.Contains(message, "already exists"),
		strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"):
		return http.StatusConflict

	case strings.Contains(message, "is required"),
		strings.Contains(message, "must contain"),
		strings.Contains(message, "provided more than once"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func writeCompanyUserResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result *usersbycompany.CompanyUserResult,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(CompanyUserHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}
