package weborder

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CountRequestsSince returns how many web order requests a company received
// since the given time (2026-10-25, security S4: daily cap of store orders).
func CountRequestsSince(companyID string, since time.Time) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var total int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_web_order_request
		WHERE company_id = $1
		  AND created_at >= $2
	`, strings.TrimSpace(companyID), since).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count web order requests: %w", err)
	}
	return total, nil
}
