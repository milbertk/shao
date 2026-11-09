package locations

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// DeleteLocation removes one location of the company. The business layer
// calls it only when the location is not used (see CountLocationUsage); the
// foreign keys also refuse it when a schedule, slot or appointment uses it.
// Returns sql.ErrNoRows when it does not exist in the company.
func DeleteLocation(companyID, locationID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	res, err := db.Exec(`
		DELETE FROM public.tbl_company_location
		WHERE location_id::text = $1
		  AND company_id = $2
	`, strings.TrimSpace(locationID), strings.TrimSpace(companyID))
	if err != nil {
		return fmt.Errorf("failed to delete location: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
