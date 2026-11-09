package locationsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HDeleteLocation deletes an unused location; a used one is disabled.
//
//	DELETE /companies/{companyId}/locations/{locationId}
func HDeleteLocation(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodDelete) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	locationID := strings.TrimSpace(mux.Vars(r)["locationId"])

	result, err := bllocations.BDeleteLocation(companyID, locationID)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while deleting the location")
		log.Printf("[HDeleteLocation] Failed: status=%d error=%v", status, err)
		return
	}
	action, message := "DeleteLocation", "Location deleted."
	if result.Disabled {
		action, message = "DisableUsedLocation", "This location is used by schedules or appointments, so it was disabled instead of deleted."
	}
	handlertracking.Record(r, action, handlertracking.Details{"locationId": locationID, "usage": result.Usage})
	writeResponse(w, http.StatusOK, message, result)
}
