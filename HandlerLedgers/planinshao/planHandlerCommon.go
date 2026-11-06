package planinshao

import (
	"encoding/json"
	"net/http"

	"github.com/milbertk/shao/bussunessLedger/blplan"
)

// PlanHandlerResponse is the standard JSON envelope for plan handlers.
type PlanHandlerResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

func writePlanResponse(w http.ResponseWriter, status int, message string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(PlanHandlerResponse{
		Status:  http.StatusText(status),
		Message: message,
		Result:  result,
	})
}

// WritePlanLimit answers 403 with { result: { code, plan, planName, limit,
// feature } } when err is a plan limit, and returns true. Other handlers
// (users, products, middleware) use it so every screen gets the same shape.
func WritePlanLimit(w http.ResponseWriter, err error) bool {
	limitErr, ok := blplan.AsPlanLimitError(err)
	if !ok {
		return false
	}
	writePlanResponse(w, http.StatusForbidden, limitErr.Error(), limitErr)
	return true
}
