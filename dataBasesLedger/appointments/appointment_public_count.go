package appointment

import (
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CountPublicRequestsSince returns how many appointment requests a company
// received from the public store since the given time (registered_by =
// 'showcase-guest') (2026-10-25, security S4: daily cap of store bookings).
func CountPublicRequestsSince(companyID string, since time.Time) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var total int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_appointment
		WHERE company_id = $1
		  AND registered_by = 'showcase-guest'
		  AND created_at >= $2
	`, strings.TrimSpace(companyID), since).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count appointment requests: %w", err)
	}
	return total, nil
}
