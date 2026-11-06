package pageconfigurationinshao

import (
	"log"
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcompanysmtp "github.com/milbertk/shao/bussunessLedger/blcompanysmtp"
)

// HSelectCompanySmtp returns the company's own SMTP configuration WITHOUT the
// password (only hasPassword).
//
//	GET /companies/{companyId}/smtp-configuration
func HSelectCompanySmtp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writePageConfigurationResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	companyID, ok := companySmtpScope(w, r)
	if !ok {
		return
	}

	result, err := blcompanysmtp.BSelectCompanySmtp(companyID)
	if err != nil {
		log.Printf("[HSelectCompanySmtp] Failed: %v", err)
		writePageConfigurationResponse(w, http.StatusInternalServerError,
			"An internal error occurred while loading the SMTP configuration", nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCompanySmtp", nil)
	writePageConfigurationResponse(w, http.StatusOK, "SMTP configuration retrieved successfully.", result)
}
