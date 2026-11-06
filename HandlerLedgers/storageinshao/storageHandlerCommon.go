package storageinshao

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/milbertk/security"

	"github.com/milbertk/shao/HandlerLedgers/planinshao"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
)

const maxStorageBodyBytes = 4 * 1024

// StorageHandlerResponse is the standard JSON envelope for storage handlers.
type StorageHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeStorageResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(StorageHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// storageContext returns the signed-in user id and the validated company.
func storageContext(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeStorageResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return "", "", false
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeStorageResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return "", "", false
	}
	return strings.TrimSpace(claims.UID), companyID, true
}

// decodeStorageBody reads one small JSON object.
func decodeStorageBody(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if r.Body == nil {
		writeStorageResponse(w, http.StatusBadRequest, "request body is required", nil)
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxStorageBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeStorageResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeStorageResponse(w, http.StatusBadRequest, "the request body must contain only one JSON object", nil)
		return false
	}
	return true
}

// writeStorageError maps business errors: plan limits (403 with
// result.code), validation (400) and the rest (500).
func writeStorageError(w http.ResponseWriter, handler string, err error) {
	if planinshao.WritePlanLimit(w, err) {
		return
	}
	if errors.Is(err, blstorage.ErrInvalidStorageFile) {
		writeStorageResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	log.Printf("[%s] failed: %v", handler, err)
	writeStorageResponse(w, http.StatusInternalServerError, "An internal error occurred", nil)
}
