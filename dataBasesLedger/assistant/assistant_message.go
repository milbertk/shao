package assistant

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// AIMessageS represents one row from public.tbl_ai_message.
type AIMessageS struct {
	MessageID      string    `json:"messageId"`
	ConversationID string    `json:"conversationId"`
	CompanyID      string    `json:"companyId"`
	UserID         string    `json:"userId"`
	Role           string    `json:"role"` // user | assistant
	Content        string    `json:"content"`
	OnTopic        *bool     `json:"onTopic,omitempty"`
	Model          *string   `json:"model,omitempty"`
	InputTokens    int       `json:"inputTokens"`
	OutputTokens   int       `json:"outputTokens"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Insert creates one message.
func (m *AIMessageS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_ai_message (
			conversation_id,
			company_id,
			user_id,
			role,
			content,
			on_topic,
			model,
			input_tokens,
			output_tokens
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING message_id, created_at
	`

	err = db.QueryRow(
		query,
		m.ConversationID,
		m.CompanyID,
		m.UserID,
		m.Role,
		m.Content,
		m.OnTopic,
		m.Model,
		m.InputTokens,
		m.OutputTokens,
	).Scan(
		&m.MessageID,
		&m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert ai message: %w", err)
	}

	return nil
}

// SelectRecentMessages returns the last `limit` messages of a conversation,
// ordered oldest -> newest (the order the Claude API expects).
func SelectRecentMessages(conversationID string, limit int) ([]AIMessageS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("conversationId is required")
	}
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT
			message_id,
			conversation_id,
			company_id,
			user_id,
			role,
			content,
			on_topic,
			model,
			input_tokens,
			output_tokens,
			created_at
		FROM (
			SELECT *
			FROM public.tbl_ai_message
			WHERE conversation_id = $1::uuid
			ORDER BY created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to select ai messages: %w", err)
	}
	defer rows.Close()

	messages := []AIMessageS{}
	for rows.Next() {
		var m AIMessageS
		if err := rows.Scan(
			&m.MessageID,
			&m.ConversationID,
			&m.CompanyID,
			&m.UserID,
			&m.Role,
			&m.Content,
			&m.OnTopic,
			&m.Model,
			&m.InputTokens,
			&m.OutputTokens,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read ai message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read ai messages: %w", err)
	}

	return messages, nil
}

// CountUserMessagesToday returns how many questions the user sent today in
// this company, and how many of them were classified as off-topic.
func CountUserMessagesToday(companyID, userID string) (total int, offTopic int, err error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, 0, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE on_topic = false)
		FROM public.tbl_ai_message
		WHERE company_id = $1
		  AND user_id = $2
		  AND role = 'user'
		  AND created_at >= date_trunc('day', now())
	`

	err = db.QueryRow(
		query,
		strings.TrimSpace(companyID),
		strings.TrimSpace(userID),
	).Scan(&total, &offTopic)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count ai messages: %w", err)
	}

	return total, offTopic, nil
}
