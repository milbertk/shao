package blcompanyverification

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/companyverification"
)

// BIsCompanyVerified reports whether the company was verified (by hand or,
// later, by the super-admin option). Without a row, or on any error, the
// answer is false: the badge is never shown by mistake.
func BIsCompanyVerified(companyID string) bool {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return false
	}

	row := companyverification.CompanyVerificationS{CompanyID: companyID}
	if err := row.SelectByCompany(); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[BIsCompanyVerified] Could not read verification: companyId=%s error=%v", companyID, err)
		}
		return false
	}
	return row.IsVerified
}
