package locationsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

type setActiveBody struct {
	Active *bool `json:"active"`
}

// HSetLocationActive enables or disables one location. Disabling with future
// appointments is allowed (they keep their location); the answer tells how
// many there are.
//
//	PUT /companies/{companyId}/locations/{locationId}/active   body: { "active": true|false }
func HSetLocationActive(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPut) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	locationID := strings.TrimSpace(mux.Vars(r)["locationId"])

	var body setActiveBody
	if err := decodeBody(r, &body); err != nil || body.Active == nil {
		writeResponse(w, http.StatusBadRequest, "body must be { \"active\": true|false }", nil)
		return
	}

	result, err := bllocations.BSetLocationActive(companyID, locationID, *body.Active)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while changing the location status")
		log.Printf("[HSetLocationActive] Failed: status=%d error=%v", status, err)
		return
	}
	action, message := "DisableLocation", "Location disabled."
	if *body.Active {
		action, message = "EnableLocation", "Location enabled."
	}
	handlertracking.Record(r, action, handlertracking.Details{"locationId": locationID, "futureAppointments": result.FutureAppointments})
	writeResponse(w, http.StatusOK, message, result)
}
