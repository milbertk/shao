package steptemplateinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blsteptemplate "github.com/milbertk/shao/bussunessLedger/blsteeptemplate"
)

// HSearchStepTemplates lists templates of the active company.
//
//	GET /companies/{companyId}/step-templates/search?term=&limit=&offset=
func HSearchStepTemplates(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchStepTemplates] Handler reached: method=%s query=%s", r.Method, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	q := r.URL.Query()
	parseInt := func(v string) int {
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	}

	// includeInactive=true (2026-10-30): the maintenance screen also lists
	// disabled processes; the product form keeps only the active ones.
	includeInactive := strings.EqualFold(strings.TrimSpace(q.Get("includeInactive")), "true")
	results, err := blsteptemplate.BSearchStepTemplatesFiltered(
		companyID, strings.TrimSpace(q.Get("term")), parseInt(q.Get("limit")), parseInt(q.Get("offset")), includeInactive,
	)
	if err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching step templates"
		}
		log.Printf("[HSearchStepTemplates] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchStepTemplates] Found %d for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchStepTemplates", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeResponse(w, http.StatusOK, "Step templates retrieved successfully.", results)
}
