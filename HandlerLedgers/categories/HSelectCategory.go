package categoriesinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
)

// HSelectCategory returns a single category of the active company by its
// internal id.
//
//	GET /companies/{companyId}/categories/{categoryInternalId}
func HSelectCategory(w http.ResponseWriter, r *http.Request) {
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

	routeVariables := mux.Vars(r)
	categoryInternalID := strings.TrimSpace(routeVariables["categoryInternalId"])
	if categoryInternalID == "" {
		writeCategoryResponse(w, http.StatusBadRequest, "categoryInternalId is required", nil)
		return
	}

	result, err := blcategories.BSelectCategoryByInternalID(companyID, categoryInternalID)
	if err != nil {
		status := categoryErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while selecting the category"
		}

		log.Printf("[HSelectCategory] Business operation failed: status=%d error=%v", status, err)
		writeCategoryResponse(w, status, message, nil)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectCategory", handlertracking.Details{"categoryInternalId": categoryInternalID})
	writeCategoryResponse(w, http.StatusOK, "Category retrieved successfully.", result)
}
