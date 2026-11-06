package customersinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blcustomers "github.com/milbertk/shao/bussunessLedger/bcostumers"
	customers "github.com/milbertk/shao/dataBasesLedger/customers"
)

// CustomerHandlerResponse is the standard JSON envelope for customer handlers.
// Result holds a *customers.CustomerS for single lookups or a
// []customers.CustomerS for searches.
type CustomerHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// writeCustomerResponse writes a JSON response with the given HTTP status.
func writeCustomerResponse(
	w http.ResponseWriter,
	status int,
	message string,
	result interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(CustomerHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// customerErrorStatus maps a business error to an HTTP status code.
func customerErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blcustomers.ErrCustomerNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "is required") {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

// decodeSingleCustomer decodes exactly one JSON object from the request body
// into a CustomerS, rejecting unknown fields and trailing content.
func decodeSingleCustomer(r *http.Request) (customers.CustomerS, error) {
	var customer customers.CustomerS

	if r.Body == nil {
		return customer, errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&customer); err != nil {
		return customer, fmt.Errorf("invalid JSON body: %w", err)
	}

	// Reject a second JSON object in the same body.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return customer, errors.New("body must contain a single JSON object")
	}

	return customer, nil
}
