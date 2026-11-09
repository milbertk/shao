package bookinginshao

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blbooking "github.com/milbertk/shao/bussunessLedger/bbooking"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// HSaveBooking creates or updates a booking configuration and rewrites its
// schedules (recurring ranges fully; manual slots from today forward).
//
//	POST /companies/{companyId}/bookings
//
// Body: a BookingBundle JSON object { config, recurringRanges, manualSlots }.
// The company is taken from the validated context, never trusted from the body.
func HSaveBooking(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveBooking] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeBookingResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
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

	bundle, err := decodeSingleBookingBundle(r)
	if err != nil {
		writeBookingResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HSaveBooking] Saving: companyId=%s bookingConfigId=%q serviceProductId=%q ranges=%d slots=%d",
		companyID,
		bundle.Config.BookingConfigID,
		bundle.Config.ServiceProductID,
		len(bundle.RecurringRanges),
		len(bundle.ManualSlots),
	)

	wasUpdate := strings.TrimSpace(bundle.Config.BookingConfigID) != ""

	result, err := blbooking.BSaveBooking(companyID, bundle)
	if exists, ok := blbooking.IsServiceScheduleExists(err); ok {
		// One schedule per service: return the existing one so the screen can
		// offer to open it (never the raw database error).
		log.Printf("[HSaveBooking] Service already has a schedule: serviceProductId=%q existingId=%s",
			bundle.Config.ServiceProductID, exists.BookingConfigID)
		writeBookingResponse(w, http.StatusConflict,
			"This service already has a booking schedule. Open it to edit it.", exists)
		return
	}
	// One location at a time (2026-11-07): the same hours are already in
	// another location. The list lets the screen show each clash.
	var overlap *bllocations.LocationOverlapError
	if errors.As(err, &overlap) {
		log.Printf("[HSaveBooking] Location overlap: %v", err)
		writeBookingResponse(w, http.StatusConflict, err.Error(), map[string]interface{}{
			"code":     "LOCATION_OVERLAP",
			"overlaps": overlap.Overlaps,
		})
		return
	}
	if err != nil {
		status := bookingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the booking"
		}

		log.Printf("[HSaveBooking] Business operation failed: status=%d error=%v", status, err)
		writeBookingResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveBooking] Saved OK: bookingConfigId=%s", result.Config.BookingConfigID)
	// Best-effort action tracking (never blocks the response).
	action := "CreateBooking"
	if wasUpdate {
		action = "UpdateBooking"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": result})
	writeBookingResponse(w, http.StatusOK, "Booking saved successfully.", result)
}
