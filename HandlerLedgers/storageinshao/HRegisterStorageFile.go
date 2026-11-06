package storageinshao

import (
	"net/http"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
)

// HRegisterStorageFile records a file the app uploaded to Firebase (product
// and company page images) so it counts in the company storage.
//
//	POST /companies/{companyId}/storage/files
//	body: { storagePath, sizeBytes, kind }   kind: PRODUCT_IMAGE | PAGE_IMAGE | OTHER
func HRegisterStorageFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeStorageResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}
	userID, companyID, ok := storageContext(w, r)
	if !ok {
		return
	}
	var body blstorage.RegisterInput
	if !decodeStorageBody(w, r, &body) {
		return
	}

	if err := blstorage.BRegisterFile(companyID, userID, body); err != nil {
		handlertracking.Record(r, "RegisterStorageFileBlocked", handlertracking.Details{
			"kind": body.Kind, "sizeBytes": body.SizeBytes, "reason": err.Error(),
		})
		writeStorageError(w, "HRegisterStorageFile", err)
		return
	}

	handlertracking.Record(r, "RegisterStorageFile", handlertracking.Details{
		"kind": body.Kind, "sizeBytes": body.SizeBytes, "storagePath": body.StoragePath,
	})
	writeStorageResponse(w, http.StatusOK, "File registered.", nil)
}
