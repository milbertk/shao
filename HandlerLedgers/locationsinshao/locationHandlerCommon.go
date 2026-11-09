package locationsinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/milbertk/security"
	"github.com/milbertk/shao/bussunessLedger/bllocations"
)

// Company locations ("Sedes", 2026-11-07).
//
//	GET    /companies/{companyId}/locations[?includeInactive=true]
//	GET    /companies/{companyId}/locations/{locationId}
//	POST   /companies/{companyId}/locations                      (create / update)
//	DELETE /companies/{companyId}/locations/{locationId}         (used -> disabled)
//	PUT    /companies/{companyId}/locations/{locationId}/active  { "active": true|false }
//	GET    /companies/{companyId}/booking-settings
//	PUT    /companies/{companyId}/booking-settings               { "locationsEnabled"?, "locationsSameTime"? }

// LocationResponse is the standard JSON envelope of these handlers.
type LocationResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(LocationResponse{Status: http.StatusText(status), Message: message, Result: result})
}

// errorStatus maps a business error to an HTTP status code.
func errorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, bllocations.ErrLocationNotFound) {
		return http.StatusNotFound
	}
	var overlap *bllocations.LocationOverlapError
	if errors.As(err, &overlap) {
		return http.StatusConflict
	}
	m := strings.ToLower(err.Error())
	switch {
	case strings.Contains(m, "duplicate"), strings.Contains(m, "unique constraint"), strings.Contains(m, "already exists"):
		return http.StatusConflict
	case strings.Contains(m, "is required"), strings.Contains(m, "invalid"), strings.Contains(m, "violates check"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// writeError answers a failed business operation; a clash of hours carries
// the list (code LOCATION_OVERLAP) so the screen can show it.
func writeError(w http.ResponseWriter, err error, internalMessage string) int {
	status := errorStatus(err)
	var overlap *bllocations.LocationOverlapError
	if errors.As(err, &overlap) {
		writeResponse(w, status, err.Error(), map[string]interface{}{
			"code":     "LOCATION_OVERLAP",
			"overlaps": overlap.Overlaps,
		})
		return status
	}
	message := err.Error()
	if status == http.StatusInternalServerError {
		message = internalMessage
	}
	writeResponse(w, status, message, nil)
	return status
}

// companyFromRequest returns the validated company of the request (never the
// body), or answers 401/403 and returns "".
func companyFromRequest(w http.ResponseWriter, r *http.Request) string {
	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return ""
	}
	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return ""
	}
	return companyID
}

// decodeBody decodes exactly one JSON object (unknown fields refused).
func decodeBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must contain a single JSON object")
	}
	return nil
}

func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeResponse(w, http.StatusMethodNotAllowed, "Use "+method, nil)
	return false
}
