package bookinginshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blbooking "github.com/milbertk/shao/bussunessLedger/bbooking"
	"github.com/milbertk/shao/dataBasesLedger/booking"
)

// HSearchBookings returns the booking configurations of the active company
// matching the optional query filters.
//
//	GET /companies/{companyId}/bookings/search
//	    ?term=&name=&serviceProductId=&availabilityMode=&limit=&offset=
func HSearchBookings(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchBookings] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

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

	query := r.URL.Query()

	parseInt := func(value string) int {
		value = strings.TrimSpace(value)
		if value == "" {
			return 0
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	}

	filters := booking.BookingConfigSearchFilters{
		Term:             strings.TrimSpace(query.Get("term")),
		Name:             strings.TrimSpace(query.Get("name")),
		ServiceProductID: strings.TrimSpace(query.Get("serviceProductId")),
		AvailabilityMode: strings.TrimSpace(query.Get("availabilityMode")),
		Limit:            parseInt(query.Get("limit")),
		Offset:           parseInt(query.Get("offset")),
	}

	results, err := blbooking.BSearchBookings(companyID, filters)
	if err != nil {
		status := bookingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching bookings"
		}

		log.Printf("[HSearchBookings] Business operation failed: status=%d error=%v", status, err)
		writeBookingResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchBookings] Found %d booking(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchBookings", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeBookingResponse(w, http.StatusOK, "Bookings retrieved successfully.", results)
}
