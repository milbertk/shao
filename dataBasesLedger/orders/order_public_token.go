package order

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// EnsurePublicToken returns the order's private "view my order" token,
// saving newToken first when it has none yet.
func EnsurePublicToken(companyID, orderInternalID, newToken string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var token string
	err = db.QueryRow(`
		UPDATE public.tbl_order
		SET public_token = COALESCE(public_token, $3)
		WHERE order_internal_id = $1 AND company_id = $2
		RETURNING public_token
	`, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID), newToken).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to set order token: %w", err)
	}
	return token, nil
}

// FindByPublicToken returns (companyId, orderInternalId) for a token, or
// sql.ErrNoRows.
func FindByPublicToken(token string) (string, string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", sql.ErrNoRows
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", "", fmt.Errorf("database connection error: %w", err)
	}
	var companyID, id string
	err = db.QueryRow(`
		SELECT company_id, order_internal_id
		FROM public.tbl_order
		WHERE public_token = $1
	`, token).Scan(&companyID, &id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", sql.ErrNoRows
		}
		return "", "", fmt.Errorf("failed to find order by token: %w", err)
	}
	return companyID, id, nil
}
