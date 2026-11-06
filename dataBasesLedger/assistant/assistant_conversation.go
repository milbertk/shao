package assistant

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// AIConversationS represents one row from public.tbl_ai_conversation.
type AIConversationS struct {
	ConversationID string    `json:"conversationId"`
	CompanyID      string    `json:"companyId"`
	UserID         string    `json:"userId"`
	Platform       string    `json:"platform"`
	Area           string    `json:"area"`
	Title          *string   `json:"title,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Insert creates one conversation.
func (c *AIConversationS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_ai_conversation (
			company_id,
			user_id,
			platform,
			area,
			title
		)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'general'), $5)
		RETURNING conversation_id, area, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		c.CompanyID,
		c.UserID,
		c.Platform,
		c.Area,
		c.Title,
	).Scan(
		&c.ConversationID,
		&c.Area,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert ai conversation: %w", err)
	}

	return nil
}

// Select loads one conversation by primary key, scoped to its company AND
// user (a user can never read another user's conversation).
func (c *AIConversationS) Select() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	conversationID := strings.TrimSpace(c.ConversationID)
	companyID := strings.TrimSpace(c.CompanyID)
	userID := strings.TrimSpace(c.UserID)
	if conversationID == "" {
		return fmt.Errorf("conversationId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}
	if userID == "" {
		return fmt.Errorf("userId is required")
	}

	query := `
		SELECT
			conversation_id,
			company_id,
			user_id,
			platform,
			area,
			title,
			created_at,
			updated_at
		FROM public.tbl_ai_conversation
		WHERE conversation_id = $1::uuid
		  AND company_id = $2
		  AND user_id = $3
	`

	err = db.QueryRow(query, conversationID, companyID, userID).Scan(
		&c.ConversationID,
		&c.CompanyID,
		&c.UserID,
		&c.Platform,
		&c.Area,
		&c.Title,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to select ai conversation: %w", err)
	}

	return nil
}

// Touch updates the conversation's updated_at (and its current area).
func (c *AIConversationS) Touch() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		UPDATE public.tbl_ai_conversation
		SET updated_at = now(),
		    area = COALESCE(NULLIF($2, ''), area)
		WHERE conversation_id = $1::uuid
		RETURNING updated_at
	`

	if err := db.QueryRow(query, c.ConversationID, c.Area).Scan(&c.UpdatedAt); err != nil {
		return fmt.Errorf("failed to update ai conversation: %w", err)
	}

	return nil
}
