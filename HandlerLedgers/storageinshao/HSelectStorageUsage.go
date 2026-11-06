package storageinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
)

// HSelectStorageUsage returns the storage the company uses and its plan limit.
//
//	GET /companies/{companyId}/storage   (any active company member)
func HSelectStorageUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeStorageResponse(w, http.StatusMethodNotAllowed, "Use GET", nil)
		return
	}
	_, companyID, ok := storageContext(w, r)
	if !ok {
		return
	}

	usage, err := blstorage.BGetStorageUsage(companyID)
	if err != nil {
		writeStorageError(w, "HSelectStorageUsage", err)
		return
	}

	// Best-effort action tracking (never blocks the response).
	handlertracking.Record(r, "SelectStorageUsage", handlertracking.Details{
		"plan": usage.PlanCode, "usedBytes": usage.UsedBytes,
	})
	writeStorageResponse(w, http.StatusOK, "Storage retrieved successfully.", usage)
}
