package weborderinshao

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
	blweborder "github.com/milbertk/shao/bussunessLedger/blweborder"
)

// HApproveWebOrder approves a web order: it creates the REAL order and sets the
// web order status to APPROVED. AUTHENTICATED (company member with rights).
//
//	POST /companies/{companyId}/web-orders/{webOrderRequestId}/approve
//
// Security: the request must belong to the active company (validated context),
// and the caller must be a member of that company. A successful approval is
// recorded with best-effort login tracking.
func HApproveWebOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HApproveWebOrder] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

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

	result, err := blweborder.BApproveWebOrder(companyID, webOrderID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, blweborder.ErrWebOrderNotFound):
			status = http.StatusNotFound
		case errors.Is(err, blweborder.ErrWebOrderAlreadyApproved):
			status = http.StatusConflict
		case strings.Contains(strings.ToLower(message), "is required"),
			errors.Is(err, blorder.ErrAppointmentInOrder): // appointments are booked, not ordered (2026-10-31)
			status = http.StatusBadRequest
		default:
			message = "An internal error occurred while approving the web order"
		}
		log.Printf("[HApproveWebOrder] Failed: status=%d error=%v", status, err)
		writeWebOrderResponse(w, status, message, nil)
		return
	}

	// Best-effort login/access tracking.
	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)
	createdOrderID := ""
	if result.CreatedOrder != nil {
		createdOrderID = result.CreatedOrder.Order.OrderInternalID
	}
	insertApproveTracking(r, actorUserID, actorEmail, companyID, actorRole, webOrderID, createdOrderID)

	log.Printf("[HApproveWebOrder] Approved OK: webOrderId=%s createdOrderId=%s", webOrderID, createdOrderID)
	writeWebOrderResponse(w, http.StatusOK, "Web order approved and order created.", result)
}

func insertApproveTracking(
	r *http.Request,
	actorUserID, actorEmail, activeCompanyID, activeCompanyRole, webOrderID, createdOrderID string,
) {
	payload := struct {
		CompanyID        string `json:"companyId"`
		ActorCompanyRole string `json:"actorCompanyRole"`
		WebOrderID       string `json:"webOrderRequestId"`
		CreatedOrderID   string `json:"createdOrderInternalId"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		WebOrderID:       webOrderID,
		CreatedOrderID:   createdOrderID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HApproveWebOrder] Error creating tracking payload: %v", err)
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
		Action:         "ApproveWebOrder",
		Jsonstring:     string(jsonPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HApproveWebOrder] Error inserting tracking: %v", err)
	}
}
