package blstorage

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/blplan"
	"github.com/milbertk/shao/dataBasesLedger/storage"
)

// Company storage (2026-10-23). Files are uploaded by the app directly to
// Firebase Storage, so shao keeps a list of them with their size
// (tbl_company_storage_file) and compares the total with the plan storage
// (tbl_plans.storage_gb, NULL = unlimited).
//
//   - Before uploading, the app asks BCheckCanStore (POST .../storage/check).
//   - After uploading, product and page images are registered with
//     BRegisterFile; order and tracking attachments are registered by shao
//     when they are saved.
//   - When the plan cannot be read or the usage cannot be summed, the upload
//     is allowed (never block the business because of a reading error).

// maxFileBytes is the biggest single file accepted (sanity limit).
const maxFileBytes int64 = 100 * 1024 * 1024

var validKinds = map[string]bool{
	storage.KindProductImage:       true,
	storage.KindPageImage:          true,
	storage.KindOrderAttachment:    true,
	storage.KindTrackingAttachment: true,
	storage.KindOther:              true,
}

// ErrInvalidStorageFile wraps validation problems (HTTP 400).
var ErrInvalidStorageFile = errors.New("invalid storage file")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidStorageFile, fmt.Sprintf(format, args...))
}

// StorageUsage is what the plans window shows.
type StorageUsage struct {
	PlanCode   string `json:"planCode"`
	UsedBytes  int64  `json:"usedBytes"`
	LimitBytes *int64 `json:"limitBytes"` // nil = unlimited
	StorageGB  *int   `json:"storageGb"`
}

// BGetStorageUsage returns the used bytes and the plan limit.
func BGetStorageUsage(companyID string) (*StorageUsage, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, invalid("companyId is required")
	}
	plan, err := blplan.BGetEffectivePlan(companyID)
	if err != nil {
		return nil, err
	}
	used, err := storage.SumUsedBytes(companyID)
	if err != nil {
		return nil, err
	}
	return &StorageUsage{
		PlanCode:   plan.PlanCode,
		UsedBytes:  used,
		LimitBytes: blplan.StorageLimitBytes(plan.Plan),
		StorageGB:  plan.Plan.StorageGB,
	}, nil
}

// BCheckCanStore returns a *blplan.PlanLimitError (PLAN_LIMIT_STORAGE or
// PLAN_REQUIRED) when extraBytes do not fit in the company plan.
func BCheckCanStore(companyID string, extraBytes int64) error {
	companyID = strings.TrimSpace(companyID)
	if extraBytes < 0 {
		return invalid("sizeBytes must be 0 or more")
	}
	if extraBytes > maxFileBytes {
		return invalid("the file is too big (maximum %d MB)", maxFileBytes/1024/1024)
	}
	if err := blplan.BCheckActivePlan(companyID); err != nil {
		return err
	}
	plan, err := blplan.BGetEffectivePlan(companyID)
	if err != nil {
		log.Printf("[blstorage] plan not read, allowing upload: companyId=%s error=%v", companyID, err)
		return nil
	}
	if blplan.StorageLimitBytes(plan.Plan) == nil {
		return nil
	}
	used, err := storage.SumUsedBytes(companyID)
	if err != nil {
		log.Printf("[blstorage] storage not summed, allowing upload: companyId=%s error=%v", companyID, err)
		return nil
	}
	return blplan.CheckStorage(plan, used, extraBytes)
}

// RegisterInput is one uploaded file.
type RegisterInput struct {
	StoragePath string `json:"storagePath"`
	SizeBytes   int64  `json:"sizeBytes"`
	Kind        string `json:"kind"`
}

func normalize(input RegisterInput) (RegisterInput, error) {
	input.StoragePath = strings.TrimSpace(input.StoragePath)
	input.Kind = strings.ToUpper(strings.TrimSpace(input.Kind))
	if input.Kind == "" {
		input.Kind = storage.KindOther
	}
	switch {
	case input.StoragePath == "":
		return input, invalid("storagePath is required")
	case len(input.StoragePath) > 500:
		return input, invalid("storagePath must have at most 500 characters")
	case input.SizeBytes < 0 || input.SizeBytes > maxFileBytes:
		return input, invalid("sizeBytes is not valid")
	case !validKinds[input.Kind]:
		return input, invalid("kind is not valid")
	}
	return input, nil
}

// BRegisterFile records an uploaded file after checking the plan storage.
func BRegisterFile(companyID, userID string, input RegisterInput) error {
	companyID = strings.TrimSpace(companyID)
	input, err := normalize(input)
	if err != nil {
		return err
	}
	// The file must be inside the company folder (2026-10-25, security S13).
	if err := ValidateCompanyStoragePath(companyID, input.StoragePath); err != nil {
		return err
	}
	// Kind aware (2026-10-29): basic company images are allowed without a plan.
	if err := checkCanStoreFile(companyID, input); err != nil {
		return err
	}
	file := storage.StorageFileS{
		CompanyID:       companyID,
		StoragePath:     input.StoragePath,
		SizeBytes:       input.SizeBytes,
		Kind:            input.Kind,
		CreatedByUserID: userID,
	}
	return file.Upsert()
}

// BForgetFile stops counting a deleted file.
func BForgetFile(companyID, storagePath string) error {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" {
		return invalid("storagePath is required")
	}
	_, err := storage.MarkDeleted(companyID, storagePath)
	return err
}

// TrackSavedFile records a file that shao already saved (order and tracking
// attachments) WITHOUT blocking: the record is already saved, so a full
// storage only shows up in the usage. Errors are logged.
func TrackSavedFile(companyID, userID, storagePath string, sizeBytes int64, kind string) {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" || sizeBytes < 0 {
		return
	}
	file := storage.StorageFileS{
		CompanyID:       strings.TrimSpace(companyID),
		StoragePath:     storagePath,
		SizeBytes:       sizeBytes,
		Kind:            kind,
		CreatedByUserID: userID,
	}
	if err := file.Upsert(); err != nil {
		log.Printf("[blstorage] file not tracked: companyId=%s path=%s error=%v", companyID, storagePath, err)
	}
}

// ForgetSavedFile stops counting a file deleted by shao (errors logged).
func ForgetSavedFile(companyID, storagePath string) {
	if strings.TrimSpace(storagePath) == "" {
		return
	}
	if _, err := storage.MarkDeleted(companyID, storagePath); err != nil {
		log.Printf("[blstorage] file not forgotten: companyId=%s path=%s error=%v", companyID, storagePath, err)
	}
}

// =====================================================================
//  Basic company images without a plan (2026-10-29).
//  A company without an active plan is read-only, but it can still set up
//  its basic company (name, logo, header) in Company page. So PAGE_IMAGE
//  files are accepted without a plan, with small limits: each image at most
//  basicCompanyMaxFileBytes and the company total at most
//  basicCompanyMaxBytes. Everything else still answers PLAN_REQUIRED.
// =====================================================================

const (
	basicCompanyMaxFileBytes int64 = 5 * 1024 * 1024  // same as the app limit
	basicCompanyMaxBytes     int64 = 20 * 1024 * 1024 // logo + header + replacements
)

// basicCompanyFolder is where the app saves the logo and the header.
func basicCompanyFolder(companyID string) string {
	return companyFolder(companyID) + "page-configuration/"
}

// isNoPlan tells if err is the "no active plan" (read-only) error.
func isNoPlan(err error) bool {
	limitErr, ok := blplan.AsPlanLimitError(err)
	return ok && limitErr.Code == blplan.CodeNoPlan
}

// checkBasicCompanyImage applies the small limits of a company without a
// plan (noPlanErr is returned when the image does not fit).
func checkBasicCompanyImage(companyID string, extraBytes int64, noPlanErr error) error {
	if extraBytes > basicCompanyMaxFileBytes {
		return invalid("the image is too big (maximum %d MB)", basicCompanyMaxFileBytes/1024/1024)
	}
	used, err := storage.SumUsedBytes(companyID)
	if err != nil {
		log.Printf("[blstorage] storage not summed, allowing basic company image: companyId=%s error=%v", companyID, err)
		return nil
	}
	if used+extraBytes > basicCompanyMaxBytes {
		return noPlanErr
	}
	return nil
}

// BCheckCanStoreKind is BCheckCanStore that also accepts the basic company
// images (kind PAGE_IMAGE) of a company without an active plan.
func BCheckCanStoreKind(companyID string, extraBytes int64, kind string) error {
	err := BCheckCanStore(companyID, extraBytes)
	if err == nil || !isNoPlan(err) {
		return err
	}
	if strings.ToUpper(strings.TrimSpace(kind)) != storage.KindPageImage {
		return err
	}
	return checkBasicCompanyImage(strings.TrimSpace(companyID), extraBytes, err)
}

// checkCanStoreFile checks a file being registered; a PAGE_IMAGE of a
// company without a plan must also be inside its page-configuration folder.
func checkCanStoreFile(companyID string, input RegisterInput) error {
	err := BCheckCanStore(companyID, input.SizeBytes)
	if err == nil || !isNoPlan(err) {
		return err
	}
	if input.Kind != storage.KindPageImage || !strings.HasPrefix(input.StoragePath, basicCompanyFolder(companyID)) {
		return err
	}
	return checkBasicCompanyImage(companyID, input.SizeBytes, err)
}
