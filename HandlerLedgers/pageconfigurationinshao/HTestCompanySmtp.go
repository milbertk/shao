package pageconfigurationinshao

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcompanysmtp "github.com/milbertk/shao/bussunessLedger/blcompanysmtp"
)

// HTestCompanySmtp sends a test email with the company's saved SMTP.
// Body (optional): { "to": "someone@example.com" }. Without "to", the
// company notification email is used.
//
//	POST /companies/{companyId}/smtp-configuration/test
func HTestCompanySmtp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writePageConfigurationResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	companyID, ok := companySmtpScope(w, r)
	if !ok {
		return
	}

	var body struct {
		To string `json:"to"`
	}
	if r.Body != nil {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 4<<10))
		if strings.TrimSpace(string(raw)) != "" {
			if err := json.Unmarshal(raw, &body); err != nil {
				writePageConfigurationResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
				return
			}
		}
	}

	result, err := blcompanysmtp.BTestCompanySmtp(companyID, body.To)
	if err != nil {
		status := pageConfigurationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while testing the SMTP configuration"
		}
		log.Printf("[HTestCompanySmtp] Failed: status=%d error=%v", status, err)
		writePageConfigurationResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "TestCompanySmtp", handlertracking.Details{
		"ok": result.OK, "sentTo": result.SentTo, "error": result.Error,
	})
	message := "Test email sent."
	if !result.OK {
		message = "The test email could not be sent."
	}
	writePageConfigurationResponse(w, http.StatusOK, message, result)
}
