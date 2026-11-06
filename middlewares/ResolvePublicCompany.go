package middlewares

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
)

// ResolvePublicCompany lets the PUBLIC store routes receive either the
// internal company id or the company's readable slug in {companyId}
// (e.g. /public/companies/maderas-finas/showcase). A slug is replaced by the
// internal id before the handler runs, so handlers, tracking and saved
// records always use the internal id.
//
// Unknown slugs pass through unchanged (the handler answers as it did
// before), so existing links never break.
func ResolvePublicCompany(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := strings.TrimSpace(vars["companyId"])

		if key != "" && !blpageconfiguration.LooksLikeCompanyID(key) {
			companyID, found, err := blpageconfiguration.BResolveCompanyKey(key)
			if err != nil {
				log.Printf("[ResolvePublicCompany] Slug lookup failed: key=%s error=%v", key, err)
			} else if found && companyID != "" {
				updated := make(map[string]string, len(vars))
				for name, value := range vars {
					updated[name] = value
				}
				updated["companyId"] = companyID
				r = mux.SetURLVars(r, updated)
			}
		}

		next.ServeHTTP(w, r)
	})
}
