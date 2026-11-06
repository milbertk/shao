package storageinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
)

type deleteStorageFileBody struct {
	StoragePath string `json:"storagePath"`
}

// HDeleteStorageFile stops counting a file the app deleted from Firebase
// (for example a replaced product image). Always allowed, also without an
// active plan, because it frees space.
//
//	POST /companies/{companyId}/storage/files/delete   body: { storagePath }
func HDeleteStorageFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeStorageResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	_, companyID, ok := storageContext(w, r)
	if !ok {
		return
	}
	var body deleteStorageFileBody
	if !decodeStorageBody(w, r, &body) {
		return
	}

	if err := blstorage.BForgetFile(companyID, body.StoragePath); err != nil {
		writeStorageError(w, "HDeleteStorageFile", err)
		return
	}

	handlertracking.Record(r, "DeleteStorageFile", handlertracking.Details{"storagePath": body.StoragePath})
	writeStorageResponse(w, http.StatusOK, "File removed from the storage count.", nil)
}
