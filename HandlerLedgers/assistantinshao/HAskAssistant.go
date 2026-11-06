package assistantinshao

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blassistant"
)

// HAskAssistant sends one question to the AI assistant (Aleph). Aleph can read
// the company's data through read-only tools.
//
//	POST /companies/{companyId}/assistant/ask
//	{ "conversationId": "", "platform": "ServiceHub", "message": "...",
//	  "currentPath": "/shao/appointments", "language": "es", "companyName": "..." }
//
// The user, company and role come from the validated context (logged-in user
// who is an active member of the company), never from the body.
func HAskAssistant(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HAskAssistant] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeAssistantResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeAssistantResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	userID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeAssistantResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	role, _ := security.GetActiveCompanyRole(r)

	item, err := decodeAssistantInput(w, r)
	if err != nil {
		writeAssistantResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HAskAssistant] Asking: companyId=%s conversationId=%q platform=%q path=%q",
		companyID, item.ConversationID, item.Platform, item.CurrentPath,
	)

	result, err := blassistant.BAskAssistant(companyID, userID, role, item)
	if err != nil {
		status := assistantErrorStatus(err)
		message := err.Error()
		var result interface{}
		switch {
		case errors.Is(err, blassistant.ErrAssistantNotInPlan):
			message = "The AI assistant is included in the PRO and ENTERPRISE plans."
			result = map[string]string{"code": "AI_NOT_IN_PLAN"}
			handlertracking.Record(r, "AskAssistantNotInPlan", handlertracking.Details{"currentPath": item.CurrentPath})
		case errors.Is(err, blassistant.ErrMonthlyInteractionsReached):
			message = "Your company used this month's AI assistant questions included in its plan. They reset on the 1st of next month; a bigger plan includes more."
			result = map[string]string{"code": "AI_MONTHLY_INTERACTIONS"}
			handlertracking.Record(r, "AskAssistantMonthlyInteractionsReached", handlertracking.Details{"currentPath": item.CurrentPath})
		case errors.Is(err, blassistant.ErrMonthlyCapReached):
			message = "Your company reached this month's AI usage limit. It resets on the 1st of next month."
			result = map[string]string{"code": "AI_MONTHLY_LIMIT"}
			handlertracking.Record(r, "AskAssistantMonthlyLimitReached", handlertracking.Details{"currentPath": item.CurrentPath})
		case status == http.StatusTooManyRequests:
			message = "You have reached today's limit of questions for the assistant. Please try again tomorrow."
			result = map[string]string{"code": "AI_DAILY_LIMIT"}
			handlertracking.Record(r, "AskAssistantLimitReached", handlertracking.Details{
				"conversationId": item.ConversationID,
				"platform":       item.Platform,
				"currentPath":    item.CurrentPath,
			})
		case status == http.StatusServiceUnavailable:
			message = "The assistant is not available right now. Please try again later."
		case status == http.StatusInternalServerError:
			message = "An internal error occurred while asking the assistant"
		}
		log.Printf("[HAskAssistant] Business operation failed: status=%d error=%v", status, err)
		writeAssistantResponse(w, status, message, result)
		return
	}

	log.Printf("[HAskAssistant] Answered OK: conversationId=%s onTopic=%t tools=%v tokens=%d/%d remaining=%d",
		result.ConversationID, result.OnTopic, result.ToolsUsed,
		result.InputTokens, result.OutputTokens, result.RemainingToday)

	// Record the successful action (best-effort). The question text is stored
	// in tbl_ai_message, so the tracking only keeps metadata.
	handlertracking.Record(r, "AskAssistant", handlertracking.Details{
		"conversationId": result.ConversationID,
		"platform":       item.Platform,
		"currentPath":    item.CurrentPath,
		"language":       item.Language,
		"onTopic":        result.OnTopic,
		"toolsUsed":      result.ToolsUsed,
		"inputTokens":    result.InputTokens,
		"outputTokens":   result.OutputTokens,
		"messageLength":  utf8.RuneCountInString(item.Message),
		"remainingToday": result.RemainingToday,
	})

	writeAssistantResponse(w, http.StatusOK, "Assistant answered successfully.", result)
}
