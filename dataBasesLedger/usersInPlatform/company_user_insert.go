package usersinplatform

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// CompanyUserS represents one row from public.tbl_company_user.
type CompanyUserS struct {
	CompanyUserID   string    `json:"companyUserId"`
	CompanyID       string    `json:"companyId"`
	UserID          string    `json:"userId"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	Active          bool      `json:"active"`
	IsProtected     bool      `json:"isProtected"`
	JoinedAt        time.Time `json:"joinedAt"`
	CreatedByUserID *string   `json:"createdByUserId,omitempty"`
	UpdatedByUserID *string   `json:"updatedByUserId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Insert creates a company-user membership.
func (a *CompanyUserS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	if a.Role == "" {
		a.Role = "operator"
	}
	if a.JoinedAt.IsZero() {
		a.JoinedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO public.tbl_company_user (
			company_id,
			user_id,
			email,
			role,
			active,
			is_protected,
			joined_at,
			created_by_user_id,
			updated_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING company_user_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
		a.UserID,
		a.Email,
		a.Role,
		a.Active,
		a.IsProtected,
		a.JoinedAt,
		a.CreatedByUserID,
		a.UpdatedByUserID,
	).Scan(
		&a.CompanyUserID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert company user: %w", err)
	}

	return nil
}
