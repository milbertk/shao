package ordertracking

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

type OrderTrackingCommentS struct {
	OrderTrackingCommentID string    `json:"orderTrackingCommentId"`
	OrderTrackingID        string    `json:"orderTrackingId"` // the stage
	CompanyID              string    `json:"companyId"`
	Author                 *string   `json:"author,omitempty"`
	Text                   string    `json:"text"`
	CommentOrder           int       `json:"commentOrder"`
	CreatedAt              time.Time `json:"createdAt"`
}

type OrderTrackingAttachmentS struct {
	OrderTrackingAttachmentID string    `json:"orderTrackingAttachmentId"`
	OrderTrackingID           string    `json:"orderTrackingId"` // the stage
	CompanyID                 string    `json:"companyId"`
	URL                       string    `json:"url"`
	StoragePath               *string   `json:"storagePath,omitempty"`
	FileName                  *string   `json:"fileName,omitempty"`
	FileType                  *string   `json:"fileType,omitempty"`
	FileSize                  int64     `json:"fileSize"`
	CreatedAt                 time.Time `json:"createdAt"`
}

func (c *OrderTrackingCommentS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	if c.CommentOrder < 1 {
		c.CommentOrder = 1
	}
	return db.QueryRow(`
		INSERT INTO public.tbl_order_tracking_comment
			(order_tracking_id, company_id, author, text, comment_order)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING order_tracking_comment_id, created_at`,
		c.OrderTrackingID, c.CompanyID, c.Author, c.Text, c.CommentOrder,
	).Scan(&c.OrderTrackingCommentID, &c.CreatedAt)
}

func (a *OrderTrackingAttachmentS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return db.QueryRow(`
		INSERT INTO public.tbl_order_tracking_attachment
			(order_tracking_id, company_id, url, storage_path, file_name, file_type, file_size)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING order_tracking_attachment_id, created_at`,
		a.OrderTrackingID, a.CompanyID, a.URL, a.StoragePath, a.FileName, a.FileType, a.FileSize,
	).Scan(&a.OrderTrackingAttachmentID, &a.CreatedAt)
}

func SelectCommentsByStage(companyID, stageID string) ([]OrderTrackingCommentS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	rows, err := db.Query(`
		SELECT order_tracking_comment_id, order_tracking_id, company_id, author, text, comment_order, created_at
		FROM public.tbl_order_tracking_comment
		WHERE order_tracking_id = $1 AND company_id = $2
		ORDER BY comment_order ASC, created_at ASC`,
		strings.TrimSpace(stageID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select stage comments: %w", err)
	}
	defer rows.Close()
	list := make([]OrderTrackingCommentS, 0)
	for rows.Next() {
		var c OrderTrackingCommentS
		if err := rows.Scan(&c.OrderTrackingCommentID, &c.OrderTrackingID, &c.CompanyID,
			&c.Author, &c.Text, &c.CommentOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func SelectAttachmentsByStage(companyID, stageID string) ([]OrderTrackingAttachmentS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	rows, err := db.Query(`
		SELECT order_tracking_attachment_id, order_tracking_id, company_id, url, storage_path, file_name, file_type, file_size, created_at
		FROM public.tbl_order_tracking_attachment
		WHERE order_tracking_id = $1 AND company_id = $2
		ORDER BY created_at ASC`,
		strings.TrimSpace(stageID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select stage attachments: %w", err)
	}
	defer rows.Close()
	list := make([]OrderTrackingAttachmentS, 0)
	for rows.Next() {
		var a OrderTrackingAttachmentS
		if err := rows.Scan(&a.OrderTrackingAttachmentID, &a.OrderTrackingID, &a.CompanyID,
			&a.URL, &a.StoragePath, &a.FileName, &a.FileType, &a.FileSize, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func DeleteCommentsByStage(companyID, stageID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`DELETE FROM public.tbl_order_tracking_comment
		WHERE order_tracking_id = $1 AND company_id = $2`,
		strings.TrimSpace(stageID), strings.TrimSpace(companyID))
	return err
}

func DeleteAttachmentsByStage(companyID, stageID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`DELETE FROM public.tbl_order_tracking_attachment
		WHERE order_tracking_id = $1 AND company_id = $2`,
		strings.TrimSpace(stageID), strings.TrimSpace(companyID))
	return err
}
