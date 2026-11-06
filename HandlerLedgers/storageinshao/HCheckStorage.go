package storageinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
)

type checkStorageBody struct {
	SizeBytes int64 `json:"sizeBytes"`
	// Optional (2026-10-29): PAGE_IMAGE lets a company without a plan upload
	// its basic company logo and header.
	Kind string `json:"kind,omitempty"`
}

// HCheckStorage tells the app if a file of sizeBytes fits in the plan
// storage BEFORE it is uploaded to Firebase. 200 = it fits; 403 with
// result.code PLAN_LIMIT_STORAGE or PLAN_REQUIRED = it does not.
//
//	POST /companies/{companyId}/storage/check   body: { sizeBytes, kind? }
func HCheckStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeStorageResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	_, companyID, ok := storageContext(w, r)
	if !ok {
		return
	}
	var body checkStorageBody
	if !decodeStorageBody(w, r, &body) {
		return
	}

	if err := blstorage.BCheckCanStoreKind(companyID, body.SizeBytes, body.Kind); err != nil {
		handlertracking.Record(r, "StorageCheckBlocked", handlertracking.Details{"sizeBytes": body.SizeBytes, "kind": body.Kind, "reason": err.Error()})
		writeStorageError(w, "HCheckStorage", err)
		return
	}

	handlertracking.Record(r, "StorageCheck", handlertracking.Details{"sizeBytes": body.SizeBytes, "kind": body.Kind})
	writeStorageResponse(w, http.StatusOK, "The file fits in your plan storage.", map[string]bool{"allowed": true})
}
