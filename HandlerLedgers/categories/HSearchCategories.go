package categoriesinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
	"github.com/milbertk/shao/dataBasesLedger/categories"
)

// HSearchCategories returns the categories of the active company matching the
// optional query filters.
//
//	GET /companies/{companyId}/categories/search
//	    ?term=&publishedCode=&categoryName=&active=&limit=&offset=
func HSearchCategories(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchCategories] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeCategoryResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeCategoryResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeCategoryResponse(w, http.StatusForbidden, "Missing validated company context", nil)
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

	filters := categories.CategorySearchFilters{
		Term:          strings.TrimSpace(query.Get("term")),
		PublishedCode: strings.TrimSpace(query.Get("publishedCode")),
		CategoryName:  strings.TrimSpace(query.Get("categoryName")),
		Limit:         parseInt(query.Get("limit")),
		Offset:        parseInt(query.Get("offset")),
	}

	// Optional active filter: only applied when explicitly provided.
	rawActive := strings.TrimSpace(query.Get("active"))
	if rawActive != "" {
		if parsed, err := strconv.ParseBool(rawActive); err == nil {
			filters.Active = &parsed
		}
	}

	results, err := blcategories.BSearchCategories(companyID, filters)
	if err != nil {
		status := categoryErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching categories"
		}

		log.Printf("[HSearchCategories] Business operation failed: status=%d error=%v", status, err)
		writeCategoryResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchCategories] Found %d category(ies) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchCategories", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeCategoryResponse(w, http.StatusOK, "Categories retrieved successfully.", results)
}
