package productsinshao

import (
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
)

// HSaveProduct creates or updates a product within a company.
//
//	POST /companies/{companyId}/products
//
// The company is taken from the validated context, never trusted from the body.
func HSaveProduct(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveProduct] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProductResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeProductResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeProductResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	product, err := decodeSingleProduct(r)
	if err != nil {
		writeProductResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	log.Printf(
		"[HSaveProduct] Saving: companyId=%s productInternalId=%q productCode=%q categoryId=%q",
		companyID, product.ProductInternalID, product.ProductCode, product.CategoryID,
	)

	wasUpdate := strings.TrimSpace(product.ProductInternalID) != ""

	result, err := blproducts.BSaveProduct(companyID, product)
	if err != nil {
		// Plan limit (2026-10-22): 403 with result.code = PLAN_LIMIT_PRODUCTS.
		if planinshao.WritePlanLimit(w, err) {
			handlertracking.Record(r, "CreateProductPlanLimit", handlertracking.Details{"error": err.Error()})
			return
		}
		status := productErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the product"
		}

		log.Printf("[HSaveProduct] Business operation failed: status=%d error=%v", status, err)
		writeProductResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSaveProduct] Saved OK: productInternalId=%s", result.ProductInternalID)
	// Best-effort action tracking (never blocks the response).
	action := "CreateProduct"
	if wasUpdate {
		action = "UpdateProduct"
	}
	handlertracking.Record(r, action, handlertracking.Details{"result": result})
	writeProductResponse(w, http.StatusOK, "Product saved successfully.", result)
}
