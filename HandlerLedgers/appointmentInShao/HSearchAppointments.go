package appointmentsinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blappointment "github.com/milbertk/shao/bussunessLedger/blappontment"
	appointment "github.com/milbertk/shao/dataBasesLedger/appointments"
)

// HSearchAppointments returns the appointments of the active company matching
// the filters.
//
//	GET /companies/{companyId}/appointments/search
//	    ?term=&customerName=&customerId=&serviceId=&status=&dateFrom=&dateTo=&limit=&offset=
func HSearchAppointments(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchAppointments] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeAppointmentResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeAppointmentResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeAppointmentResponse(w, http.StatusForbidden, "Missing validated company context", nil)
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

	filters := appointment.AppointmentSearchFilters{
		Term:         strings.TrimSpace(query.Get("term")),
		CustomerName: strings.TrimSpace(query.Get("customerName")),
		CustomerID:   strings.TrimSpace(query.Get("customerId")),
		ServiceID:    strings.TrimSpace(query.Get("serviceId")),
		LocationID:   strings.TrimSpace(query.Get("locationId")),
		Status:       strings.TrimSpace(query.Get("status")),
		DateFrom:     strings.TrimSpace(query.Get("dateFrom")),
		DateTo:       strings.TrimSpace(query.Get("dateTo")),
		Limit:        parseInt(query.Get("limit")),
		Offset:       parseInt(query.Get("offset")),
	}

	results, err := blappointment.BSearchAppointments(companyID, filters)
	if err != nil {
		status := appointmentErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching appointments"
		}
		log.Printf("[HSearchAppointments] Business operation failed: status=%d error=%v", status, err)
		writeAppointmentResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchAppointments] Found %d appointment(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchAppointments", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeAppointmentResponse(w, http.StatusOK, "Appointments retrieved successfully.", results)
}
