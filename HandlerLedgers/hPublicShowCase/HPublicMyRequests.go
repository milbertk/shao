package publicshowcaseinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
)

// HPublicMyRequests returns "My orders / My bookings" of the SIGNED-IN store
// customer for one company.
//
//	GET /public/companies/{companyId}/my-requests   (companyId = id or slug)
//
// It runs after security.RequireActiveUser (valid session cookie), so the
// user and email come from the validated token, never from the request.
func HPublicMyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeShowcaseResponse(w, http.StatusUnauthorized, "Please sign in", nil)
		return
	}

	companyID := strings.TrimSpace(mux.Vars(r)["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
		return
	}

	view, err := blpublicshowcase.BMyRequests(companyID, claims.UID, claims.Email, claims.EmailVerified)
	if err != nil {
		log.Printf("[HPublicMyRequests] Business operation failed: companyId=%s error=%v", companyID, err)
		writeShowcaseResponse(w, http.StatusInternalServerError,
			"An internal error occurred while loading your orders and bookings", nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicMyRequests", companyID, handlertracking.Details{
		"orders":        len(view.Orders),
		"bookings":      len(view.Bookings),
		"emailVerified": claims.EmailVerified,
	})
	writeShowcaseResponse(w, http.StatusOK, "Your orders and bookings.", view)
}
