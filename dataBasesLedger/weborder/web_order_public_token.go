package weborder

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// EnsurePublicToken returns the web order's private "manage my request"
// token, saving newToken first when it has none yet.
func EnsurePublicToken(companyID, webOrderRequestID, newToken string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	var token string
	err = db.QueryRow(`
		UPDATE public.tbl_web_order_request
		SET public_token = COALESCE(public_token, $3)
		WHERE web_order_request_id = $1 AND company_id = $2
		RETURNING public_token
	`, strings.TrimSpace(webOrderRequestID), strings.TrimSpace(companyID), newToken).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to set web order token: %w", err)
	}
	return token, nil
}

// FindByPublicToken returns (companyId, webOrderRequestId) for a token, or
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
		SELECT company_id, web_order_request_id
		FROM public.tbl_web_order_request
		WHERE public_token = $1
	`, token).Scan(&companyID, &id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", sql.ErrNoRows
		}
		return "", "", fmt.Errorf("failed to find web order by token: %w", err)
	}
	return companyID, id, nil
}
