package locationsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HSaveLocation creates (no locationId) or updates one location.
//
//	POST /companies/{companyId}/locations
//	body: { locationId?, name, address?, details?, latitude?, longitude?, active? }
func HSaveLocation(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	var input bllocations.LocationInput
	if err := decodeBody(r, &input); err != nil {
		writeResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	wasUpdate := strings.TrimSpace(input.LocationID) != ""

	saved, err := bllocations.BSaveLocation(companyID, input)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while saving the location")
		log.Printf("[HSaveLocation] Failed: status=%d error=%v", status, err)
		return
	}
	action := "CreateLocation"
	if wasUpdate {
		action = "UpdateLocation"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": saved})
	writeResponse(w, http.StatusOK, "Location saved.", saved)
}
