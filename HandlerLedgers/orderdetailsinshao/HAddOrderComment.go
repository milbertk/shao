package orderdetailsinshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	blorderdetails "github.com/milbertk/shao/bussunessLedger/borderdetails"
	"github.com/milbertk/tracking"
)

type addCommentBody struct {
	CommentText string `json:"commentText"`
}

// HAddOrderComment adds a comment to the whole order.
//
//	POST /companies/{companyId}/orders/{orderInternalId}/comments
//	Body: { commentText }
func HAddOrderComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeODResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeODResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeODResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}
	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeODResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return
	}
	if r.Body == nil {
		writeODResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	var body addCommentBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeODResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeODResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	actorEmail, _ := security.GetActiveCompanyEmail(r)
	result, err := blorderdetails.BAddOrderComment(companyID, orderID, body.CommentText, actorUserID, actorEmail)
	if err != nil {
		status := odErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while adding the comment"
		}
		log.Printf("[HAddOrderComment] Failed: status=%d error=%v", status, err)
		writeODResponse(w, status, msg, nil)
		return
	}
	insertODTracking(r, actorUserID, actorEmail, companyID, orderID, "AddOrderComment")
	writeODResponse(w, http.StatusOK, "Comment added successfully.", result)
}

func insertODTracking(r *http.Request, userID, email, companyID, orderID, action string) {
	payload := struct {
		CompanyID       string `json:"companyId"`
		OrderInternalID string `json:"orderInternalId"`
	}{companyID, orderID}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[%s] Error creating tracking payload: %v", action, err)
		return
	}
	lt := tracking.LoginTracking{
		UserID:         userID,
		Email:          strings.TrimSpace(email),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readODClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         action,
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[%s] Error inserting tracking: %v", action, err)
	}
}
