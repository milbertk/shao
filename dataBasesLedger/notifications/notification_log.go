package notifications

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// NotificationLogS is one row of public.tbl_notification_log: one email that
// was sent, failed or skipped (e.g. no email address, SMTP not configured).
type NotificationLogS struct {
	CompanyID    string
	EntityType   string // WEB_ORDER | ORDER | APPOINTMENT | SMTP_TEST
	EntityID     string
	Event        string
	Audience     string // CUSTOMER | COMPANY
	Recipient    string
	Status       string // SENT | FAILED | SKIPPED
	ErrorMessage string
}

func nilIfEmpty(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// Insert saves one log row.
func (a *NotificationLogS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	errorMessage := a.ErrorMessage
	if len(errorMessage) > 1000 {
		errorMessage = errorMessage[:1000]
	}

	_, err = db.Exec(`
		INSERT INTO public.tbl_notification_log (
			company_id, entity_type, entity_id, event, audience,
			recipient, status, error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		strings.TrimSpace(a.CompanyID),
		a.EntityType,
		a.EntityID,
		a.Event,
		a.Audience,
		nilIfEmpty(a.Recipient),
		a.Status,
		nilIfEmpty(errorMessage),
	)
	if err != nil {
		return fmt.Errorf("failed to insert notification log: %w", err)
	}
	return nil
}

// LastSentAt returns when an event was last SENT for one record and audience
// (nil when it was never sent). Used, for example, to know if a customer
// already sent a cancellation request from their link.
func LastSentAt(companyID, entityType, entityID, event, audience string) (*time.Time, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	var last sql.NullTime
	err = db.QueryRow(`
		SELECT MAX(created_at)
		FROM public.tbl_notification_log
		WHERE company_id = $1
		  AND entity_type = $2
		  AND entity_id = $3
		  AND event = $4
		  AND audience = $5
		  AND status = 'SENT'
	`,
		strings.TrimSpace(companyID),
		entityType,
		strings.TrimSpace(entityID),
		event,
		audience,
	).Scan(&last)
	if err != nil {
		return nil, fmt.Errorf("failed to read notification log: %w", err)
	}
	if !last.Valid {
		return nil, nil
	}
	return &last.Time, nil
}

// CountSentToRecipientSince returns how many emails were SENT to one address
// (any company) since the given time (2026-10-25, security S4: daily cap of
// customer emails, so the store forms cannot be used to flood a stranger).
func CountSentToRecipientSince(recipient string, since time.Time) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var total int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_notification_log
		WHERE lower(recipient) = lower($1)
		  AND audience = 'CUSTOMER'
		  AND status = 'SENT'
		  AND created_at >= $2
	`, strings.TrimSpace(recipient), since).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count emails to recipient: %w", err)
	}
	return total, nil
}
