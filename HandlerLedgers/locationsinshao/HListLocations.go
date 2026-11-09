package locationsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HListLocations returns the locations of the company (with usage).
//
//	GET /companies/{companyId}/locations[?includeInactive=true]
func HListLocations(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	includeInactive := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("includeInactive")), "true")

	list, err := bllocations.BListLocations(companyID, includeInactive)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while loading the locations")
		log.Printf("[HListLocations] Failed: status=%d error=%v", status, err)
		return
	}
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "ListLocations", handlertracking.Details{"includeInactive": includeInactive, "resultCount": len(list)})
	writeResponse(w, http.StatusOK, "Locations loaded.", list)
}
