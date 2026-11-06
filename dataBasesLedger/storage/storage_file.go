package storage

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Storage kinds (tbl_company_storage_file.kind).
const (
	KindProductImage       = "PRODUCT_IMAGE"
	KindPageImage          = "PAGE_IMAGE"
	KindOrderAttachment    = "ORDER_ATTACHMENT"
	KindTrackingAttachment = "TRACKING_ATTACHMENT"
	KindOther              = "OTHER"
)

// StorageFileS is one row of public.tbl_company_storage_file: one file a
// company uploaded to Firebase Storage, with its size (2026-10-23).
// The files are uploaded by the app directly to Firebase; this table is how
// shao knows how much storage each company uses.
type StorageFileS struct {
	CompanyID       string
	StoragePath     string
	SizeBytes       int64
	Kind            string
	CreatedByUserID string
}

func nilIfEmpty(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// Upsert records a file (or updates its size). A file deleted before and
// uploaded again with the same path counts again.
func (a *StorageFileS) Upsert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO public.tbl_company_storage_file (
			company_id, storage_path, size_bytes, kind, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (company_id, storage_path) DO UPDATE
		   SET size_bytes = EXCLUDED.size_bytes,
		       kind       = EXCLUDED.kind,
		       deleted_at = NULL
	`,
		strings.TrimSpace(a.CompanyID),
		strings.TrimSpace(a.StoragePath),
		a.SizeBytes,
		a.Kind,
		nilIfEmpty(a.CreatedByUserID),
	)
	if err != nil {
		return fmt.Errorf("failed to save storage file: %w", err)
	}
	return nil
}

// MarkDeleted stops counting a file (its row is kept with deleted_at).
func MarkDeleted(companyID, storagePath string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	result, err := db.Exec(`
		UPDATE public.tbl_company_storage_file
		   SET deleted_at = now()
		 WHERE company_id = $1 AND storage_path = $2 AND deleted_at IS NULL
	`, strings.TrimSpace(companyID), strings.TrimSpace(storagePath))
	if err != nil {
		return 0, fmt.Errorf("failed to delete storage file: %w", err)
	}
	return result.RowsAffected()
}

// SumUsedBytes returns the bytes the company uses now.
func SumUsedBytes(companyID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	var used int64
	err = db.QueryRow(`
		SELECT COALESCE(SUM(size_bytes), 0)
		FROM public.tbl_company_storage_file
		WHERE company_id = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(companyID)).Scan(&used)
	if err != nil {
		return 0, fmt.Errorf("failed to sum storage: %w", err)
	}
	return used, nil
}
