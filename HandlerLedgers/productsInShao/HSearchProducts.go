package productsinshao

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// HSearchProducts returns the products of the active company matching the
// optional query filters.
//
//	GET /companies/{companyId}/products/search
//	    ?term=&productCode=&name=&categoryId=&productType=&status=&limit=&offset=
func HSearchProducts(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSearchProducts] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeProductResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
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

	filters := products.ProductSearchFilters{
		Term:        strings.TrimSpace(query.Get("term")),
		ProductCode: strings.TrimSpace(query.Get("productCode")),
		Name:        strings.TrimSpace(query.Get("name")),
		CategoryID:  strings.TrimSpace(query.Get("categoryId")),
		ProductType: strings.TrimSpace(query.Get("productType")),
		Status:      strings.TrimSpace(query.Get("status")),
		Limit:       parseInt(query.Get("limit")),
		Offset:      parseInt(query.Get("offset")),
	}

	results, err := blproducts.BSearchProducts(companyID, filters)
	if err != nil {
		status := productErrorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while searching products"
		}

		log.Printf("[HSearchProducts] Business operation failed: status=%d error=%v", status, err)
		writeProductResponse(w, status, message, nil)
		return
	}

	log.Printf("[HSearchProducts] Found %d product(s) for companyId=%s", len(results), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SearchProducts", handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(results)})
	writeProductResponse(w, http.StatusOK, "Products retrieved successfully.", results)
}
