package locationsinshao

import (
	"log"
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HSelectBookingSettings returns the company switches "We use locations"
// (2026-11-08) and "We can attend in more than one location at the same time".
//
//	GET /companies/{companyId}/booking-settings
func HSelectBookingSettings(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	settings, err := bllocations.BGetBookingSettings(companyID)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while loading the booking settings")
		log.Printf("[HSelectBookingSettings] Failed: status=%d error=%v", status, err)
		return
	}
	handlertracking.Record(r, "SelectBookingSettings", nil)
	writeResponse(w, http.StatusOK, "Booking settings loaded.", settings)
}

// HSaveBookingSettings saves the switches (one or both). A change that starts
// the "one location at a time" rule is refused (409, LOCATION_OVERLAP) while
// schedules have the same hours in two locations.
//
//	PUT /companies/{companyId}/booking-settings
//	body: { "locationsEnabled"?: true|false, "locationsSameTime"?: true|false }
func HSaveBookingSettings(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPut) {
		return
	}
	companyID := companyFromRequest(w, r)
	if companyID == "" {
		return
	}
	var input bllocations.BookingSettingsInput
	if err := decodeBody(r, &input); err != nil {
		writeResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	settings, err := bllocations.BSaveBookingSettings(companyID, input)
	if err != nil {
		status := writeError(w, err, "An internal error occurred while saving the booking settings")
		log.Printf("[HSaveBookingSettings] Failed: status=%d error=%v", status, err)
		return
	}
	handlertracking.Record(r, "SaveBookingSettings", handlertracking.Details{
		"locationsEnabled":  settings.LocationsEnabled,
		"locationsSameTime": settings.LocationsSameTime,
	})
	writeResponse(w, http.StatusOK, "Booking settings saved.", settings)
}
