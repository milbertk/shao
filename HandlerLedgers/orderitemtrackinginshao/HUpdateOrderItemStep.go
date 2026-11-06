package orderitemtrackinginshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blorderitemtracking "github.com/milbertk/shao/bussunessLedger/blorderitemtracking"
)

type updateItemStepBody struct {
	OrderItemID       string `json:"orderItemId"`
	CurrentStepID     string `json:"currentStepId"`
	CurrentStepStatus string `json:"currentStepStatus"`
}

// HUpdateOrderItemStep updates one item's step and recomputes the order status.
//
//	POST /companies/{companyId}/orders/{orderInternalId}/item-tracking
//	Body: { orderItemId, currentStepId, currentStepStatus }
func HUpdateOrderItemStep(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HUpdateOrderItemStep] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeOITResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeOITResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeOITResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeOITResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}

	if r.Body == nil {
		writeOITResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	var body updateItemStepBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeOITResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeOITResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	actorEmail, _ := security.GetActiveCompanyEmail(r)

	result, err := blorderitemtracking.BUpdateOrderItemStep(
		companyID, orderID, body.OrderItemID, body.CurrentStepID, body.CurrentStepStatus,
		actorUserID, actorEmail,
	)
	if err != nil {
		status := oitErrorStatus(err.Error())
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while updating the item step"
		}
		log.Printf("[HUpdateOrderItemStep] Failed: status=%d error=%v", status, err)
		writeOITResponse(w, status, message, nil)
		return
	}

	// Best-effort login tracking.
	actorRole, _ := security.GetActiveCompanyRole(r)
	insertItemStepTracking(r, actorUserID, actorEmail, companyID, actorRole, orderID, body.OrderItemID, result.OrderStatus)

	log.Printf("[HUpdateOrderItemStep] Updated OK: order=%s item=%s newOrderStatus=%s",
		orderID, body.OrderItemID, result.OrderStatus)
	writeOITResponse(w, http.StatusOK, "Item step updated successfully.", result)
}

func insertItemStepTracking(
	r *http.Request,
	actorUserID, actorEmail, activeCompanyID, activeCompanyRole, orderID, orderItemID, newOrderStatus string,
) {
	payload := struct {
		CompanyID        string `json:"companyId"`
		ActorCompanyRole string `json:"actorCompanyRole"`
		OrderInternalID  string `json:"orderInternalId"`
		OrderItemID      string `json:"orderItemId"`
		NewOrderStatus   string `json:"newOrderStatus"`
	}{activeCompanyID, activeCompanyRole, orderID, orderItemID, newOrderStatus}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HUpdateOrderItemStep] Error creating tracking payload: %v", err)
		return
	}
	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readOITClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         "UpdateOrderItemStep",
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[HUpdateOrderItemStep] Error inserting tracking: %v", err)
	}
}
