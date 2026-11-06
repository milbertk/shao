package publicshowcaseinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blassistant"
)

const maxPublicAssistantBodyBytes = 16 * 1024

// HPublicAssistant answers one visitor question on a company's public
// showcase (Aleph, shopping assistant). PUBLIC (no authentication). The company
// comes from the URL and scopes every search. It only reads the public
// catalog; it returns "actions" the page runs on screen (add to cart, open
// booking, show contact...). It NEVER places orders or booking requests.
//
//	POST /public/companies/{companyId}/assistant/ask
//	{ "conversationId": "", "message": "...", "visitorId": "...",
//	  "companyName": "...", "language": "es", "cart": [{ "productId": "...", "quantity": 1 }] }
func HPublicAssistant(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HPublicAssistant] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	companyID := strings.TrimSpace(mux.Vars(r)["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}

	if r.Body == nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPublicAssistantBodyBytes)

	var req blassistant.ShowcaseAskInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeShowcaseResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	// Signed-in visitor (same rule as the public web order request); otherwise
	// the random visitor id kept by the page; otherwise the IP.
	loggedUserID := ""
	if claims, ok := security.GetClaims(r); ok && claims != nil {
		loggedUserID = strings.TrimSpace(claims.UID)
	} else if claims, err := security.ReadClaimsFromCookie(r); err == nil && claims != nil {
		loggedUserID = strings.TrimSpace(claims.UID)
	}
	clientIP := handlertracking.ClientIP(r)
	visitorKey := blassistant.ShowcaseVisitorKey(loggedUserID, req.VisitorID, clientIP)

	result, err := blassistant.BAskShowcaseAssistant(companyID, visitorKey, clientIP, loggedUserID != "", req)
	if err != nil {
		status, message := publicAssistantError(err)
		log.Printf("[HPublicAssistant] Failed: status=%d companyId=%s error=%v", status, companyID, err)
		if status == http.StatusTooManyRequests {
			handlertracking.RecordPublic(r, "PublicAskAssistantLimitReached", companyID, handlertracking.Details{
				"conversationId": req.ConversationID,
			})
		}
		writeShowcaseResponse(w, status, message, nil)
		return
	}

	actionTypes := make([]string, 0, len(result.Actions))
	for _, action := range result.Actions {
		actionTypes = append(actionTypes, action.Type)
	}
	log.Printf("[HPublicAssistant] Answered OK: companyId=%s conversationId=%s actions=%v tokens=%d/%d",
		companyID, result.ConversationID, actionTypes, result.InputTokens, result.OutputTokens)

	// Best-effort action tracking (never blocks the response). Only metadata:
	// the question text is stored in tbl_ai_message.
	handlertracking.RecordPublic(r, "PublicAskAssistant", companyID, handlertracking.Details{
		"conversationId": result.ConversationID,
		"actions":        actionTypes,
		"onTopic":        result.OnTopic,
		"inputTokens":    result.InputTokens,
		"outputTokens":   result.OutputTokens,
		"messageLength":  utf8.RuneCountInString(req.Message),
		"cartLines":      len(req.Cart),
		"language":       req.Language,
	})

	writeShowcaseResponse(w, http.StatusOK, "Assistant answered successfully.", result)
}

func publicAssistantError(err error) (int, string) {
	switch {
	case errors.Is(err, blassistant.ErrShowcaseNotFound):
		return http.StatusNotFound, "This shop is not available."
	case errors.Is(err, blassistant.ErrConversationNotFound):
		return http.StatusNotFound, "conversation not found"
	case errors.Is(err, blassistant.ErrAssistantNotInPlan):
		return http.StatusForbidden, "The assistant is not available for this shop."
	case errors.Is(err, blassistant.ErrMonthlyCapReached):
		return http.StatusTooManyRequests, "The assistant is resting for now. You can keep browsing and buying normally."
	case errors.Is(err, blassistant.ErrDailyLimitReached),
		errors.Is(err, blassistant.ErrShowcaseBusy):
		return http.StatusTooManyRequests, "The assistant has reached today's limit. Please try again tomorrow."
	case errors.Is(err, blassistant.ErrAssistantUnavailable):
		return http.StatusServiceUnavailable, "The assistant is not available right now. Please try again later."
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "is required") || strings.Contains(message, "invalid") {
		return http.StatusBadRequest, err.Error()
	}
	return http.StatusInternalServerError, "An internal error occurred while asking the assistant"
}
