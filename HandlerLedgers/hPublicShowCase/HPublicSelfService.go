package publicshowcaseinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
)

// Customer self-service links ("manage my request"). PUBLIC: the long random
// token in the URL is the only key. Responses never include internal ids of
// the appointment/order. Limited per IP per day.
//
//	GET  /public/self-service/{token}
//	POST /public/self-service/{token}/cancel-request  { reason }  (email to the company; manual cancel)
//	POST /public/self-service/{token}/reschedule      { appointmentDate, startHour, endHour }

const (
	selfServiceViewsPerIPPerDay   = 300
	selfServiceActionsPerIPPerDay = 30
)

type ipDayCounter struct {
	mu     sync.Mutex
	day    string
	counts map[string]int
}

func (c *ipDayCounter) allow(key string, limit int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	today := time.Now().Format("2006-01-02")
	if c.day != today || c.counts == nil || len(c.counts) > 100000 {
		c.day, c.counts = today, map[string]int{}
	}
	if c.counts[key] >= limit {
		return false
	}
	c.counts[key]++
	return true
}

var (
	selfServiceViewCounter   = &ipDayCounter{}
	selfServiceActionCounter = &ipDayCounter{}
)

func selfServiceToken(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["token"])
}

// writeSelfServiceError maps business errors to HTTP answers with a code the
// store page can translate.
func writeSelfServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, blpublicshowcase.ErrSelfServiceNotFound):
		writeShowcaseResponse(w, http.StatusNotFound, "This link is not valid.", map[string]string{"code": "NOT_FOUND"})
	case errors.Is(err, blpublicshowcase.ErrSelfServiceNotAllowed):
		writeShowcaseResponse(w, http.StatusConflict, "This action is not possible anymore.", map[string]string{"code": "NOT_ALLOWED"})
	case errors.Is(err, blpublicshowcase.ErrSlotTaken):
		writeShowcaseResponse(w, http.StatusConflict, "This time was just taken. Please choose another time.", map[string]string{"code": "SLOT_TAKEN"})
	case errors.Is(err, blpublicshowcase.ErrSameTime):
		writeShowcaseResponse(w, http.StatusBadRequest, "Choose a different time from the current one.", map[string]string{"code": "SAME_TIME"})
	case errors.Is(err, blpublicshowcase.ErrCancelAlreadyRequested):
		writeShowcaseResponse(w, http.StatusConflict, "A cancellation request was already sent.", map[string]string{"code": "ALREADY_REQUESTED"})
	case errors.Is(err, blpublicshowcase.ErrCancelUnavailable):
		writeShowcaseResponse(w, http.StatusServiceUnavailable, "The business can not receive cancellation requests right now. Please contact them directly.", map[string]string{"code": "CANCEL_UNAVAILABLE"})
	case errors.Is(err, blpublicshowcase.ErrServiceNotBookable):
		writeShowcaseResponse(w, http.StatusBadRequest, "This service does not accept online bookings.", map[string]string{"code": "SERVICE_NOT_BOOKABLE"})
	default:
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "is required") || strings.HasPrefix(low, "invalid") {
			writeShowcaseResponse(w, http.StatusBadRequest, err.Error(), map[string]string{"code": "BAD_REQUEST"})
			return
		}
		log.Printf("[HPublicSelfService] internal error: %v", err)
		writeShowcaseResponse(w, http.StatusInternalServerError, "An internal error occurred. Please try again.", map[string]string{"code": "INTERNAL"})
	}
}

func tooManyRequests(w http.ResponseWriter) {
	writeShowcaseResponse(w, http.StatusTooManyRequests, "Too many requests. Please try again later.", map[string]string{"code": "RATE_LIMIT"})
}

// HPublicSelfServiceView shows the customer's appointment / order.
func HPublicSelfServiceView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	if !selfServiceViewCounter.allow(readPublicClientIP(r), selfServiceViewsPerIPPerDay) {
		tooManyRequests(w)
		return
	}

	view, err := blpublicshowcase.BSelfServiceView(selfServiceToken(r))
	if err != nil {
		writeSelfServiceError(w, err)
		return
	}

	handlertracking.RecordPublic(r, "PublicSelfServiceView", view.CompanyID, handlertracking.Details{
		"kind": view.Kind, "reference": view.Reference, "status": view.Status,
	})
	w.Header().Set("Cache-Control", "no-store")
	writeShowcaseResponse(w, http.StatusOK, "OK", view)
}

// HPublicSelfServiceCancelRequest sends a cancellation REQUEST to the
// company by email. Nothing is cancelled: the manager cancels manually.
func HPublicSelfServiceCancelRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	if !selfServiceActionCounter.allow(readPublicClientIP(r), selfServiceActionsPerIPPerDay) {
		tooManyRequests(w)
		return
	}

	// The reason is optional: an empty body is accepted.
	var input blpublicshowcase.CancelRequestInput
	if r.Body != nil {
		decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			writeShowcaseResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
			return
		}
	}

	view, err := blpublicshowcase.BSelfServiceRequestCancel(selfServiceToken(r), input)
	if err != nil {
		writeSelfServiceError(w, err)
		return
	}

	handlertracking.RecordPublic(r, "PublicSelfServiceCancelRequest", view.CompanyID, handlertracking.Details{
		"kind": view.Kind, "reference": view.Reference, "status": view.Status,
		"hasReason": strings.TrimSpace(input.Reason) != "",
	})
	writeShowcaseResponse(w, http.StatusOK, "Cancellation request sent.", view)
}

// HPublicSelfServiceReschedule moves an appointment to a new free time; it
// goes back to PENDING so the company confirms it.
func HPublicSelfServiceReschedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	if !selfServiceActionCounter.allow(readPublicClientIP(r), selfServiceActionsPerIPPerDay) {
		tooManyRequests(w)
		return
	}

	var input blpublicshowcase.RescheduleInput
	if r.Body == nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}

	view, err := blpublicshowcase.BSelfServiceReschedule(selfServiceToken(r), input)
	if err != nil {
		writeSelfServiceError(w, err)
		return
	}

	handlertracking.RecordPublic(r, "PublicSelfServiceReschedule", view.CompanyID, handlertracking.Details{
		"kind": view.Kind, "date": input.AppointmentDate, "startHour": input.StartHour, "endHour": input.EndHour,
	})
	writeShowcaseResponse(w, http.StatusOK, "New time requested.", view)
}
