package pageconfigurationinshao

import (
	"log"
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcompanysmtp "github.com/milbertk/shao/bussunessLedger/blcompanysmtp"
)

// HSaveCompanySmtp creates or replaces the company's own SMTP. An empty
// password keeps the one already saved. The password is stored encrypted and
// is never written to the tracking details.
//
//	PUT /companies/{companyId}/smtp-configuration
func HSaveCompanySmtp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writePageConfigurationResponse(w, http.StatusMethodNotAllowed, "Use PUT", nil)
		return
	}

	companyID, ok := companySmtpScope(w, r)
	if !ok {
		return
	}

	input, err := decodeCompanySmtp(r)
	if err != nil {
		writePageConfigurationResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	result, err := blcompanysmtp.BSaveCompanySmtp(companyID, input)
	if err != nil {
		status := pageConfigurationErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the SMTP configuration"
		}
		log.Printf("[HSaveCompanySmtp] Failed: status=%d error=%v", status, err)
		writePageConfigurationResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking. The result has no password.
	handlertracking.Record(r, "SaveCompanySmtp", handlertracking.Details{
		"host": result.Host, "port": result.Port, "security": result.Security,
		"fromEmail": result.FromEmail, "enabled": result.Enabled,
		"passwordChanged": input.Password != "",
	})
	writePageConfigurationResponse(w, http.StatusOK, "SMTP configuration saved successfully.", result)
}
