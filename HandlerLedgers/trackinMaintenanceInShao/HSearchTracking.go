package trackinginshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	bltracking "github.com/milbertk/shao/bussunessLedger/bltracking"
	tracking "github.com/milbertk/shao/dataBasesLedger/trackingsteeps"
)

// HSearchTracking returns the tracking headers of the active company matching
// the optional query filters.
//
//	GET /companies/{companyId}/trackings/search
//	    ?term=&name=&internalCode=&targetType=&categoryId=&productId=&limit=&offset=
func HSearchTracking(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchTracking] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeTrackingResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeTrackingResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeTrackingResponse(w, http.StatusForbidden, "Missing validated company context", nil)
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

	filters := tracking.TrackingConfigSearchFilters{
		Term:         strings.TrimSpace(query.Get("term")),
		Name:         strings.TrimSpace(query.Get("name")),
		InternalCode: strings.TrimSpace(query.Get("internalCode")),
		TargetType:   strings.TrimSpace(query.Get("targetType")),
		CategoryID:   strings.TrimSpace(query.Get("categoryId")),
		ProductID:    strings.TrimSpace(query.Get("productId")),
		Limit:        parseInt(query.Get("limit")),
		Offset:       parseInt(query.Get("offset")),
	}

	results, err := bltracking.BSearchTracking(companyID, filters)
	if err != nil {
		status := trackingErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching trackings"
		}

		log.Printf("[HSearchTracking] Business operation failed: status=%d error=%v", status, err)
		writeTrackingResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchTracking] Found %d tracking(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchTrackingConfigs", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeTrackingResponse(w, http.StatusOK, "Trackings retrieved successfully.", results)
}
