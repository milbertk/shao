package publicshowcaseinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
	"github.com/milbertk/tracking"

	"github.com/milbertk/security"
)

// HPublicWebOrderRequest saves a guest web order (cart) as PENDING and sends a
// notification email. PUBLIC (no authentication). Company comes from the URL.
//
//	POST /public/companies/{companyId}/web-order-requests
//
// Body: PublicWebOrderRequest JSON { orderType, useRegisteredCustomer,
//
//	customerId, customerName, phone, email, socialLink, preferredContact,
//	contactValue, region, address, notes, items: [...] }.
//
// A successful save is recorded with best-effort access/login tracking.
func HPublicWebOrderRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HPublicWebOrderRequest] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	companyID := strings.TrimSpace(mux.Vars(r)["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}

	if r.Body == nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "request body is required", nil)
		return
	}

	// Spam / email-flood protection (2026-10-25, security S4): body size,
	// per-IP and per-company daily limits, before anything is saved or sent.
	r.Body = http.MaxBytesReader(w, r.Body, maxPublicFormBodyBytes)
	if !checkPublicFormLimits(w, r, blpublicshowcase.PublicFormWebOrder, companyID) {
		return
	}
	// CAPTCHA (Cloudflare Turnstile, 2026-10-25): bots cannot send the form.
	if !checkPublicCaptcha(w, r, blpublicshowcase.PublicFormWebOrder, companyID) {
		return
	}

	var req blpublicshowcase.PublicWebOrderRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeShowcaseResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeShowcaseResponse(w, http.StatusBadRequest, "body must contain a single JSON object", nil)
		return
	}

	// Optional session: if a valid user is logged in, capture their code so the
	// order is stored as a PLATFORM_USER. On a public visit (guest) this stays
	// empty and the order is stored as GUEST. The code comes from the validated
	// session, never from the request body.
	// Signed-in store customer? The public route has no auth middleware, so
	// read the session cookie directly (validated). Guests stay anonymous.
	userCode := ""
	if claims, ok := security.GetClaims(r); ok && claims != nil {
		userCode = strings.TrimSpace(claims.UID)
	} else if claims, err := security.ReadClaimsFromCookie(r); err == nil && claims != nil {
		userCode = strings.TrimSpace(claims.UID)
	}

	result, err := blpublicshowcase.BCreatePublicWebOrderRequest(companyID, userCode, req)
	if err != nil {
		message := err.Error()
		status := http.StatusInternalServerError
		low := strings.ToLower(message)
		if isPublicFormInvalid(err) || errors.Is(err, blorder.ErrAppointmentInOrder) ||
			strings.Contains(low, "is required") || strings.Contains(low, "requires") {
			status = http.StatusBadRequest
		} else if strings.Contains(low, "duplicate") || strings.Contains(low, "unique constraint") {
			status = http.StatusConflict
		} else {
			message = "An internal error occurred while saving the web order"
		}
		log.Printf("[HPublicWebOrderRequest] Failed: status=%d error=%v", status, err)
		writeShowcaseResponse(w, status, message, nil)
		return
	}

	log.Printf("[HPublicWebOrderRequest] Saved OK: code=%s id=%s",
		result.RequestCode, result.WebOrderRequestID)

	// Best-effort access/login tracking (public visitor, no auth session).
	insertWebOrderAccessTracking(r, companyID, result.RequestCode, result.CustomerName)

	writeShowcaseResponse(w, http.StatusOK, "Your web order request was sent.", result)
}

// insertWebOrderAccessTracking records the public web-order submission into the
// login tracking table. Best-effort: a failure is logged, never blocks.
func insertWebOrderAccessTracking(r *http.Request, companyID, requestCode, customerName string) {
	payload := struct {
		CompanyID    string `json:"companyId"`
		RequestCode  string `json:"requestCode"`
		CustomerName string `json:"customerName"`
	}{
		CompanyID:    companyID,
		RequestCode:  requestCode,
		CustomerName: customerName,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HPublicWebOrderRequest] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         "public-guest",
		Email:          "",
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readPublicClientIP(r),
		Platform:       "ShaoShowcase",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         "PublicWebOrderRequest",
		Jsonstring:     string(jsonPayload),
	}

	if err := lt.Insert(); err != nil {
		log.Printf("[HPublicWebOrderRequest] Error inserting tracking: %v", err)
	}
}

// readPublicClientIP returns the visitor IP. It uses handlertracking.ClientIP,
// which trusts X-Forwarded-For only behind your own proxy (TRUSTED_PROXY_HOPS),
// so a visitor cannot fake the IP to skip the limits (2026-10-25, security S4).
func readPublicClientIP(r *http.Request) string {
	return handlertracking.ClientIP(r)
}
