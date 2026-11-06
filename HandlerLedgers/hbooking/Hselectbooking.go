package bookinginshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blbooking "github.com/milbertk/shao/bussunessLedger/bbooking"
)

// HSelectBooking returns one booking with its header + recurring ranges +
// manual slots (the full bundle), scoped to the active company.
//
//	GET /companies/{companyId}/bookings/{bookingConfigId}
func HSelectBooking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeBookingResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeBookingResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeBookingResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	routeVariables := mux.Vars(r)
	bookingConfigID := strings.TrimSpace(routeVariables["bookingConfigId"])
	if bookingConfigID == "" {
		writeBookingResponse(w, http.StatusBadRequest, "bookingConfigId is required", nil)
		return
	}

	result, err := blbooking.BSelectBookingFull(companyID, bookingConfigID)
	if err != nil {
		status := bookingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the booking"
		}

		log.Printf("[HSelectBooking] Business operation failed: status=%d error=%v", status, err)
		writeBookingResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectBooking", handlertracking.Details{"bookingConfigId": bookingConfigID})
	writeBookingResponse(w, http.StatusOK, "Booking retrieved successfully.", result)
}
