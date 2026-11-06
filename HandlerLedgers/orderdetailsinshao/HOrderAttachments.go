package orderdetailsinshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	blorderdetails "github.com/milbertk/shao/bussunessLedger/borderdetails"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
)

type addAttachmentsBody struct {
	Attachments []struct {
		URL         string `json:"url"`
		StoragePath string `json:"storagePath"`
		FileName    string `json:"fileName"`
		FileType    string `json:"fileType"`
		FileSize    int64  `json:"fileSize"`
	} `json:"attachments"`
}

// HAddOrderAttachments records already-uploaded Firebase files on the order.
//
//	POST /companies/{companyId}/orders/{orderInternalId}/attachments
//	Body: { attachments: [{ url, storagePath, fileName, fileType, fileSize }] }
func HAddOrderAttachments(w http.ResponseWriter, r *http.Request) {
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
	var body addAttachmentsBody
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

	files := make([]order.OrderAttachmentRow, 0, len(body.Attachments))
	for _, a := range body.Attachments {
		files = append(files, order.OrderAttachmentRow{
			URL: a.URL, StoragePath: a.StoragePath, FileName: a.FileName,
			FileType: a.FileType, FileSize: a.FileSize,
		})
	}

	actorEmail, _ := security.GetActiveCompanyEmail(r)
	result, err := blorderdetails.BAddOrderAttachments(companyID, orderID, files, actorUserID, actorEmail)
	if err != nil {
		// Plan storage full / no active plan (2026-10-23): 403 with result.code.
		if planinshao.WritePlanLimit(w, err) {
			insertODTracking(r, actorUserID, actorEmail, companyID, orderID, "AddOrderAttachmentsPlanBlocked")
			return
		}
		status := odErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while saving the attachments"
		}
		log.Printf("[HAddOrderAttachments] Failed: status=%d error=%v", status, err)
		writeODResponse(w, status, msg, nil)
		return
	}
	insertODTracking(r, actorUserID, actorEmail, companyID, orderID, "AddOrderAttachments")
	writeODResponse(w, http.StatusOK, "Attachments saved successfully.", result)
}

// HDeleteOrderComment removes one comment.
//
//	DELETE /companies/{companyId}/orders/{orderInternalId}/comments/{commentId}
func HDeleteOrderComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeODResponse(w, http.StatusMethodNotAllowed, "Use DELETE", nil)
		return
	}
	companyID, orderID, ok := odAuthScope(w, r)
	if !ok {
		return
	}
	commentID := strings.TrimSpace(mux.Vars(r)["commentId"])
	result, err := blorderdetails.BDeleteOrderComment(companyID, orderID, commentID)
	if err != nil {
		status := odErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while deleting the comment"
		}
		log.Printf("[HDeleteOrderComment] Failed: status=%d error=%v", status, err)
		writeODResponse(w, status, msg, nil)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "DeleteOrderComment", handlertracking.Details{"orderInternalId": orderID, "commentId": commentID})
	writeODResponse(w, http.StatusOK, "Comment deleted successfully.", result)
}

// HDeleteOrderAttachment removes one attachment (DB row). The Firebase object is
// deleted by the frontend using the returned storagePath, or you can wire a
// server-side Firebase admin delete here if preferred.
//
//	DELETE /companies/{companyId}/orders/{orderInternalId}/attachments/{attachmentId}
func HDeleteOrderAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeODResponse(w, http.StatusMethodNotAllowed, "Use DELETE", nil)
		return
	}
	companyID, orderID, ok := odAuthScope(w, r)
	if !ok {
		return
	}
	attachmentID := strings.TrimSpace(mux.Vars(r)["attachmentId"])
	result, storagePath, err := blorderdetails.BDeleteOrderAttachment(companyID, orderID, attachmentID)
	if err != nil {
		status := odErrorStatus(err.Error())
		msg := err.Error()
		if status == http.StatusInternalServerError {
			msg = "An internal error occurred while deleting the attachment"
		}
		log.Printf("[HDeleteOrderAttachment] Failed: status=%d error=%v", status, err)
		writeODResponse(w, status, msg, nil)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "DeleteOrderAttachment", handlertracking.Details{"orderInternalId": orderID, "attachmentId": attachmentID, "storagePath": storagePath})
	writeODResponse(w, http.StatusOK, "Attachment deleted successfully.", map[string]interface{}{
		"details":     result,
		"storagePath": storagePath,
	})
}

// odAuthScope validates claims + company + order id for the delete handlers.
func odAuthScope(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeODResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return "", "", false
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeODResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return "", "", false
	}
	orderID := strings.TrimSpace(mux.Vars(r)["orderInternalId"])
	if orderID == "" {
		writeODResponse(w, http.StatusBadRequest, "orderInternalId is required", nil)
		return "", "", false
	}

	return companyID, orderID, true
}
