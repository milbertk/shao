package order

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderAttachmentS represents one row from public.tbl_order_attachment.
type OrderAttachmentS struct {
	OrderAttachmentID string    `json:"orderAttachmentId"`
	OrderInternalID   string    `json:"orderInternalId"`
	CompanyID         string    `json:"companyId"`
	URL               string    `json:"url"`
	StoragePath       *string   `json:"storagePath,omitempty"`
	FileName          *string   `json:"fileName,omitempty"`
	FileType          *string   `json:"fileType,omitempty"`
	FileSize          int64     `json:"fileSize"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Insert creates one order attachment reference.
func (a *OrderAttachmentS) InsertWith(db dbmanager.Executor) error {
	var err error

	query := `
		INSERT INTO public.tbl_order_attachment (
			order_internal_id,
			company_id,
			url,
			storage_path,
			file_name,
			file_type,
			file_size
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING order_attachment_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.OrderInternalID,
		a.CompanyID,
		a.URL,
		a.StoragePath,
		a.FileName,
		a.FileType,
		a.FileSize,
	).Scan(
		&a.OrderAttachmentID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order attachment: %w", err)
	}

	return nil
}

// Insert creates one order attachment reference (own connection).
func (a *OrderAttachmentS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return a.InsertWith(db)
}

const orderAttachmentColumns = `
	order_attachment_id,
	order_internal_id,
	company_id,
	url,
	storage_path,
	file_name,
	file_type,
	file_size,
	created_at,
	updated_at
`

func scanOrderAttachment(row interface {
	Scan(dest ...any) error
}, a *OrderAttachmentS) error {
	return row.Scan(
		&a.OrderAttachmentID,
		&a.OrderInternalID,
		&a.CompanyID,
		&a.URL,
		&a.StoragePath,
		&a.FileName,
		&a.FileType,
		&a.FileSize,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
}

// SelectAttachmentsByOrder returns every attachment of an order, scoped to the
// company.
func SelectAttachmentsByOrder(companyID, orderInternalID string) ([]OrderAttachmentS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}
	if orderInternalID == "" {
		return nil, fmt.Errorf("orderInternalId is required")
	}

	query := `
		SELECT ` + orderAttachmentColumns + `
		FROM public.tbl_order_attachment
		WHERE order_internal_id = $1
		  AND company_id = $2
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, orderInternalID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to select order attachments: %w", err)
	}
	defer rows.Close()

	list := make([]OrderAttachmentS, 0)
	for rows.Next() {
		var item OrderAttachmentS
		if err := scanOrderAttachment(rows, &item); err != nil {
			return nil, fmt.Errorf("failed to scan order attachment: %w", err)
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading order attachments: %w", err)
	}

	return list, nil
}

// DeleteAttachmentsByOrder removes ALL attachment references of an order,
// scoped to the company (used by the rewrite-on-save flow). Note: this removes
// the DB rows only; deleting the actual Firebase files is done client-side.
func DeleteAttachmentsByOrderWith(db dbmanager.Executor, companyID, orderInternalID string) (int64, error) {
	var err error

	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" {
		return 0, fmt.Errorf("companyId is required")
	}
	if orderInternalID == "" {
		return 0, fmt.Errorf("orderInternalId is required")
	}

	result, err := db.Exec(`
		DELETE FROM public.tbl_order_attachment
		WHERE order_internal_id = $1
		  AND company_id = $2
	`, orderInternalID, companyID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete order attachments: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted order attachments: %w", err)
	}

	return deleted, nil
}

// DeleteAttachmentsByOrder removes every attachment reference of an order
// (own connection).
func DeleteAttachmentsByOrder(companyID, orderInternalID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	return DeleteAttachmentsByOrderWith(db, companyID, orderInternalID)
}
