package ordersinshao

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	blorder "github.com/milbertk/shao/bussunessLedger/blorder"
)

type OrderHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writeOrderResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(OrderHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

func orderErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, blorder.ErrOrderNotFound) {
		return http.StatusNotFound
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "duplicate"),
		strings.Contains(message, "unique constraint"),
		strings.Contains(message, "already exists"):
		return http.StatusConflict
	case strings.Contains(message, "is required"),
		strings.Contains(message, "invalid"),
		strings.Contains(message, "violates check"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeSingleOrderBundle(r *http.Request) (blorder.OrderBundle, error) {
	var bundle blorder.OrderBundle
	if r.Body == nil {
		return bundle, errors.New("request body is required")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return bundle, fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return bundle, errors.New("body must contain a single JSON object")
	}
	return bundle, nil
}
