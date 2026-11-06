package order

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// -------------------------------------------------------------------
//  Order-level comments
// -------------------------------------------------------------------

type OrderCommentRow struct {
	OrderCommentID string `json:"orderCommentId"`
	CommentText    string `json:"commentText"`
	AuthorUserID   string `json:"authorUserId"`
	AuthorEmail    string `json:"authorEmail"`
	CreatedAt      string `json:"createdAt"`
}

func SelectOrderComments(companyID, orderInternalID string) ([]OrderCommentRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	rows, err := db.Query(`
		SELECT order_comment_id, comment_text,
		       COALESCE(author_user_id, ''), COALESCE(author_email, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS')
		FROM public.tbl_order_comment
		WHERE order_internal_id = $1 AND company_id = $2
		ORDER BY created_at ASC
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select order comments: %w", err)
	}
	defer rows.Close()
	out := make([]OrderCommentRow, 0)
	for rows.Next() {
		var c OrderCommentRow
		if err := rows.Scan(&c.OrderCommentID, &c.CommentText, &c.AuthorUserID, &c.AuthorEmail, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order comment: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func InsertOrderComment(companyID, orderInternalID, text, userID, email string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var id string
	err = db.QueryRow(`
		INSERT INTO public.tbl_order_comment
			(order_internal_id, company_id, comment_text, author_user_id, author_email)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING order_comment_id
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID),
		strings.TrimSpace(text), strings.TrimSpace(userID), strings.TrimSpace(email)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert order comment: %w", err)
	}
	return id, nil
}

func DeleteOrderComment(companyID, orderInternalID, commentID string) (int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	res, err := db.Exec(`
		DELETE FROM public.tbl_order_comment
		WHERE order_comment_id = $1 AND order_internal_id = $2 AND company_id = $3
	`, strings.TrimSpace(commentID), strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID))
	if err != nil {
		return 0, fmt.Errorf("failed to delete order comment: %w", err)
	}
	return res.RowsAffected()
}

// -------------------------------------------------------------------
//  Order-level attachments
// -------------------------------------------------------------------

type OrderAttachmentRow struct {
	OrderAttachmentID string `json:"orderAttachmentId"`
	URL               string `json:"url"`
	StoragePath       string `json:"storagePath"`
	FileName          string `json:"fileName"`
	FileType          string `json:"fileType"`
	FileSize          int64  `json:"fileSize"`
	UploadedByUserID  string `json:"uploadedByUserId"`
	UploadedByEmail   string `json:"uploadedByEmail"`
	CreatedAt         string `json:"createdAt"`
}

func SelectOrderAttachments(companyID, orderInternalID string) ([]OrderAttachmentRow, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	rows, err := db.Query(`
		SELECT order_attachment_id, url, COALESCE(storage_path, ''),
		       COALESCE(file_name, ''), COALESCE(file_type, ''), file_size,
		       COALESCE(uploaded_by_user_id, ''), COALESCE(uploaded_by_email, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS')
		FROM public.tbl_order_attachment
		WHERE order_internal_id = $1 AND company_id = $2
		ORDER BY created_at ASC
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select order attachments: %w", err)
	}
	defer rows.Close()
	out := make([]OrderAttachmentRow, 0)
	for rows.Next() {
		var a OrderAttachmentRow
		if err := rows.Scan(&a.OrderAttachmentID, &a.URL, &a.StoragePath, &a.FileName,
			&a.FileType, &a.FileSize, &a.UploadedByUserID, &a.UploadedByEmail, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order attachment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func InsertOrderAttachment(companyID, orderInternalID string, a OrderAttachmentRow, userID, email string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var id string
	err = db.QueryRow(`
		INSERT INTO public.tbl_order_attachment
			(order_internal_id, company_id, url, storage_path, file_name, file_type, file_size,
			 uploaded_by_user_id, uploaded_by_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING order_attachment_id
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID),
		strings.TrimSpace(a.URL), strings.TrimSpace(a.StoragePath), strings.TrimSpace(a.FileName),
		strings.TrimSpace(a.FileType), a.FileSize, strings.TrimSpace(userID), strings.TrimSpace(email)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert order attachment: %w", err)
	}
	return id, nil
}

// DeleteOrderAttachment removes the DB row and returns its storage_path so the
// caller can delete the Firebase object.
func DeleteOrderAttachment(companyID, orderInternalID, attachmentID string) (string, int64, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", 0, fmt.Errorf("database connection error: %w", err)
	}
	var storagePath string
	err = db.QueryRow(`
		DELETE FROM public.tbl_order_attachment
		WHERE order_attachment_id = $1 AND order_internal_id = $2 AND company_id = $3
		RETURNING COALESCE(storage_path, '')
	`, strings.TrimSpace(attachmentID), strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).Scan(&storagePath)
	if err != nil {
		// No row deleted → not found (QueryRow returns ErrNoRows).
		return "", 0, fmt.Errorf("failed to delete order attachment: %w", err)
	}
	return storagePath, 1, nil
}
