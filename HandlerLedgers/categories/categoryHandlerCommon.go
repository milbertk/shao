package categoriesinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
	"github.com/milbertk/shao/dataBasesLedger/categories"
)

// CategoryHandlerResponse is the standard JSON envelope for category handlers.
// Result holds a *categories.CategoryS for single lookups or a
// []categories.CategoryS for searches.
type CategoryHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// writeCategoryResponse writes a JSON response with the given HTTP status.
func writeCategoryResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(CategoryHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// categoryErrorStatus maps a business error to an HTTP status code.
func categoryErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blcategories.ErrCategoryNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())

	switch {
	// published_code is unique per company: a duplicate is a conflict.
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"):
		return http.StatusConflict

	case strings.Contains(message, "is required"),
		strings.Contains(message, "must contain"),
		strings.Contains(message, "not blank"),
		strings.Contains(message, "invalid"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// decodeSingleCategory decodes exactly one JSON object from the request body
// into a CategoryS, rejecting unknown fields and trailing content.
func decodeSingleCategory(r *http.Request) (categories.CategoryS, error) {
	var category categories.CategoryS

	if r.Body == nil {
		return category, errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&category); err != nil {
		return category, fmt.Errorf("invalid JSON body: %w", err)
	}

	// Reject a second JSON object in the same body.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return category, errors.New("body must contain a single JSON object")
	}

	return category, nil
}
