package weborderinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blweborder "github.com/milbertk/shao/bussunessLedger/blweborder"
)

type updateStatusBody struct {
	Status string `json:"status"`
}

// HUpdateWebOrderStatus updates a web order's status (PENDING/APPROVED/REJECTED).
//
//	POST /companies/{companyId}/web-orders/{webOrderRequestId}/status
//	Body: { "status": "APPROVED" }
//
// A successful update is recorded with best-effort login tracking.
func HUpdateWebOrderStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HUpdateWebOrderStatus] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeWebOrderResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeWebOrderResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeWebOrderResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	webOrderID := strings.TrimSpace(mux.Vars(r)["webOrderRequestId"])
	if webOrderID == "" {
		writeWebOrderResponse(w, http.StatusBadRequest, "webOrderRequestId is required", nil)
		return
	}

	if r.Body == nil {
		writeWebOrderResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	var body updateStatusBody
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeWebOrderResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeWebOrderResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	result, err := blweborder.BUpdateWebOrderStatus(companyID, webOrderID, body.Status)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, blweborder.ErrWebOrderNotFound) {
			status = http.StatusNotFound
		} else if strings.Contains(strings.ToLower(message), "invalid") ||
			strings.Contains(strings.ToLower(message), "is required") {
			status = http.StatusBadRequest
		} else {
			message = "An internal error occurred while updating the web order"
		}
		log.Printf("[HUpdateWebOrderStatus] Failed: status=%d error=%v", status, err)
		writeWebOrderResponse(w, status, message, nil)
		return
	}

	// Best-effort login/access tracking.
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)
	insertWebOrderStatusTracking(r, actorUserID, actorEmail, companyID, actorRole, webOrderID, strings.ToUpper(strings.TrimSpace(body.Status)))

	log.Printf("[HUpdateWebOrderStatus] Updated OK: id=%s status=%s", webOrderID, body.Status)
	writeWebOrderResponse(w, http.StatusOK, "Web order status updated successfully.", result)
}

func insertWebOrderStatusTracking(
	r *http.Request,
	actorUserID, actorEmail, activeCompanyID, activeCompanyRole, webOrderID, status string,
) {
	payload := struct {
		CompanyID        string `json:"companyId"`
		ActorCompanyRole string `json:"actorCompanyRole"`
		WebOrderID       string `json:"webOrderRequestId"`
		NewStatus        string `json:"newStatus"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		WebOrderID:       webOrderID,
		NewStatus:        status,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HUpdateWebOrderStatus] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readWebOrderClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         "UpdateWebOrderStatus",
		Jsonstring:     string(jsonPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HUpdateWebOrderStatus] Error inserting tracking: %v", err)
	}
}

func readWebOrderClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	return strings.TrimSpace(r.RemoteAddr)
}
