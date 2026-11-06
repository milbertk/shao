package publicshowcaseinshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpublicshowcase "github.com/milbertk/shao/bussunessLedger/bPublicShowcase"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// PublicShowcaseResponse is the standard JSON envelope for the public showcase.
type PublicShowcaseResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeShowcaseResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(PublicShowcaseResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// HPublicShowcase returns the PUBLIC catalog of a company: the products that
// are visible on the web and active. It is UNAUTHENTICATED — the company comes
// from the URL, not from a session. It is READ-ONLY (GET) and returns only
// public-safe fields.
//
//	GET /public/companies/{companyId}/showcase
//	    ?term=&category=&productType=&id=&limit=&offset=
//
//	- Empty criteria  -> all public products for the company.
//	- With criteria   -> filtered by term (name/description), category,
//	                     productType, or id.
func HPublicShowcase(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HPublicShowcase] Handler reached: method=%s path=%s query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeShowcaseResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}

	routeVariables := mux.Vars(r)
	companyID := strings.TrimSpace(routeVariables["companyId"])
	if companyID == "" {
		writeShowcaseResponse(w, http.StatusBadRequest, "companyId is required", nil)
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

	filters := products.PublicProductSearchFilters{
		CompanyID:   companyID,
		Term:        strings.TrimSpace(query.Get("term")),
		CategoryID:  strings.TrimSpace(query.Get("category")),
		ProductType: strings.TrimSpace(query.Get("productType")),
		ProductID:   strings.TrimSpace(query.Get("id")),
		Limit:       parseInt(query.Get("limit")),
		Offset:      parseInt(query.Get("offset")),
	}

	items, err := blpublicshowcase.BSearchShowcase(companyID, filters)
	if err != nil {
		message := err.Error()
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(message), "is required") {
			status = http.StatusBadRequest
		} else {
			message = "An internal error occurred while loading the showcase"
		}
		log.Printf("[HPublicShowcase] Operation failed: status=%d error=%v", status, err)
		writeShowcaseResponse(w, status, message, nil)
		return
	}

	log.Printf("[HPublicShowcase] Returned %d item(s) for companyId=%s", len(items), companyID)
	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicViewShowcase", companyID, handlertracking.Details{"filters": handlertracking.SearchFilters(r), "resultCount": len(items)})
	writeShowcaseResponse(w, http.StatusOK, "Showcase retrieved successfully.", items)
}
