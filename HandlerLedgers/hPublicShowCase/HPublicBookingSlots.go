package publicshowcaseinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
)

// PublicBookingSlotsResponse is the JSON envelope for the slots endpoint.
type PublicBookingSlotsResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeSlotsResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(PublicBookingSlotsResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// HPublicBookingSlots returns the bookable slots for a company + product on a
// given date. PUBLIC and READ-ONLY (no authentication).
//
//	GET /public/companies/{companyId}/products/{productId}/slots?date=YYYY-MM-DD
//
// The result includes the mode (RECURRING/MANUAL), whether the day is
// available, and the slots that still have remaining capacity.
func HPublicBookingSlots(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HPublicBookingSlots] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeSlotsResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	routeVariables := mux.Vars(r)
	companyID := strings.TrimSpace(routeVariables["companyId"])
	productID := strings.TrimSpace(routeVariables["productId"])
	date := strings.TrimSpace(r.URL.Query().Get("date"))

	if companyID == "" {
		writeSlotsResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}
	if productID == "" {
		writeSlotsResponse(w, http.StatusBadRequest, "productId is required", nil)
		return
	}
	if date == "" {
		writeSlotsResponse(w, http.StatusBadRequest, "date is required (YYYY-MM-DD)", nil)
		return
	}

	result, err := blpublicshowcase.BGetAvailabilityForDate(companyID, productID, date)
	if err != nil {
		message := err.Error()
		status := http.StatusInternalServerError
		low := strings.ToLower(message)
		if strings.Contains(low, "is required") || strings.Contains(low, "invalid date") {
			status = http.StatusBadRequest
		} else {
			message = "An internal error occurred while loading the slots"
		}
		log.Printf("[HPublicBookingSlots] Operation failed: status=%d error=%v", status, err)
		writeSlotsResponse(w, status, message, nil)
		return
	}

	log.Printf("[HPublicBookingSlots] date=%s mode=%s available=%t slots=%d",
		result.Date, result.Mode, result.Available, len(result.Slots))
	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicViewBookingSlots", companyID, handlertracking.Details{"productId": productID, "date": date, "slotCount": len(result.Slots)})
	writeSlotsResponse(w, http.StatusOK, "Slots retrieved successfully.", result)
}
