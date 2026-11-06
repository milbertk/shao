package productsinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blproducts "github.com/milbertk/shao/bussunessLedger/products"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// ProductHandlerResponse is the standard JSON envelope for product handlers.
type ProductHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeProductResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(ProductHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// productErrorStatus maps a business error to an HTTP status code.
func productErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blproducts.ErrProductNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())

	switch {
	// product_code is unique per company: a duplicate is a conflict.
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"):
		return http.StatusConflict

	case strings.Contains(message, "does not exist in this company"),
		strings.Contains(message, "is required"),
		strings.Contains(message, "must contain"),
		strings.Contains(message, "not blank"),
		strings.Contains(message, "invalid"):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

// decodeSingleProduct decodes exactly one JSON object from the request body
// into a ProductS, rejecting unknown fields and trailing content.
func decodeSingleProduct(r *http.Request) (products.ProductS, error) {
	var product products.ProductS

	if r.Body == nil {
		return product, errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&product); err != nil {
		return product, fmt.Errorf("invalid JSON body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return product, errors.New("body must contain a single JSON object")
	}

	return product, nil
}
