package companyverification

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyVerificationS represents one row from public.tbl_company_verification.
// It is read-only from shao for now; a super-admin option will edit it later.
type CompanyVerificationS struct {
	CompanyVerificationID string     `json:"companyVerificationId"`
	CompanyID             string     `json:"companyId"`
	IsVerified            bool       `json:"isVerified"`
	VerifiedAt            *time.Time `json:"verifiedAt,omitempty"`
	VerifiedBy            *string    `json:"-"`
	Notes                 *string    `json:"-"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

// SelectByCompany loads the verification row of CompanyID.
// Returns a wrapped sql.ErrNoRows when the company has none.
func (a *CompanyVerificationS) SelectByCompany() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	companyID := strings.TrimSpace(a.CompanyID)
	if companyID == "" {
		return errors.New("companyId is required")
	}

	query := `
		SELECT
			company_verification_id,
			company_id,
			is_verified,
			verified_at,
			verified_by,
			notes,
			created_at,
			updated_at
		FROM public.tbl_company_verification
		WHERE company_id = $1
	`
	err = db.QueryRow(query, companyID).Scan(
		&a.CompanyVerificationID,
		&a.CompanyID,
		&a.IsVerified,
		&a.VerifiedAt,
		&a.VerifiedBy,
		&a.Notes,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("company verification not found: %w", err)
		}
		return fmt.Errorf("failed to select company verification: %w", err)
	}
	return nil
}
