package assistant

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CountCompanyPlatformMessagesToday returns how many questions were sent today
// to one company on one platform (for example "showcase"), from all visitors.
// Used as a daily budget for the public assistant.
func CountCompanyPlatformMessagesToday(companyID, platform string) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}

	query := `
		SELECT COUNT(*)
		FROM public.tbl_ai_message m
		JOIN public.tbl_ai_conversation c ON c.conversation_id = m.conversation_id
		WHERE m.company_id = $1
		  AND c.platform = $2
		  AND m.role = 'user'
		  AND m.created_at >= date_trunc('day', now())
	`

	var total int
	err = db.QueryRow(
		query,
		strings.TrimSpace(companyID),
		strings.TrimSpace(platform),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count company ai messages: %w", err)
	}

	return total, nil
}
