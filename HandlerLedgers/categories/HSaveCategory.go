package categoriesinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
)

// HSaveCategory creates or updates a category within a company.
//
//	POST /companies/{companyId}/categories
//
// The company is taken from the validated context (set by
// RequireActiveCompanyMember), never trusted from the body.
func HSaveCategory(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveCategory] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeCategoryResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
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

	category, err := decodeSingleCategory(r)
	if err != nil {
		writeCategoryResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HSaveCategory] Saving: companyId=%s categoryInternalId=%q publishedCode=%q",
		companyID, category.CategoryInternalID, category.PublishedCode,
	)

	wasUpdate := strings.TrimSpace(category.CategoryInternalID) != ""

	result, err := blcategories.BSaveCategory(companyID, category)
	if err != nil {
		status := categoryErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the category"
		}

		log.Printf("[HSaveCategory] Business operation failed: status=%d error=%v", status, err)
		writeCategoryResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveCategory] Saved OK: categoryInternalId=%s", result.CategoryInternalID)
	// Best-effort action tracking (never blocks the response).
	action := "CreateCategory"
	if wasUpdate {
		action = "UpdateCategory"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": result})
	writeCategoryResponse(w, http.StatusOK, "Category saved successfully.", result)
}
