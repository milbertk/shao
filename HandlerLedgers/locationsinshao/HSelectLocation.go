package locationsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HSelectLocation returns one location of the company.
//
//	GET /companies/{companyId}/locations/{locationId}
func HSelectLocation(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	locationID := strings.TrimSpace(mux.Vars(r)["locationId"])

	location, err := bllocations.BSelectLocation(companyID, locationID)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while loading the location")
		log.Printf("[HSelectLocation] Failed: status=%d error=%v", status, err)
		return
	}
	handlertracking.Record(r, "SelectLocation", handlertracking.Details{"locationId": locationID})
	writeResponse(w, http.StatusOK, "Location loaded.", location)
}
