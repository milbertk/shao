package planrequests

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// PlanRequestS is one row of public.tbl_plan_request.
type PlanRequestS struct {
	PlanRequestID     string    `json:"planRequestId"`
	CompanyID         string    `json:"companyId"`
	CompanyName       string    `json:"companyName"`
	CurrentPlan       string    `json:"currentPlan"`
	RequestedPlan     string    `json:"requestedPlan"`
	ContactName       string    `json:"contactName"`
	ContactEmail      string    `json:"contactEmail"`
	ContactPhone      string    `json:"contactPhone"`
	Message           string    `json:"message"`
	Language          string    `json:"language"`
	Status            string    `json:"status"`
	RequestedByUserID string    `json:"-"`
	EmailSent         bool      `json:"emailSent"`
	CreatedAt         time.Time `json:"createdAt"`
}

// Insert saves a new request (status PENDING) and fills its id and date.
func (a *PlanRequestS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	a.Status = "PENDING"
	err = db.QueryRow(`
		INSERT INTO public.tbl_plan_request (
			company_id, company_name, current_plan, requested_plan,
			contact_name, contact_email, contact_phone, message, language,
			status, requested_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, NULLIF($11, ''))
		RETURNING plan_request_id::text, created_at
	`,
		a.CompanyID, a.CompanyName, a.CurrentPlan, a.RequestedPlan,
		a.ContactName, a.ContactEmail, a.ContactPhone, a.Message, a.Language,
		a.Status, a.RequestedByUserID,
	).Scan(&a.PlanRequestID, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save plan request: %w", err)
	}
	return nil
}

// MarkEmailSent records that the request email reached the SMTP server.
func MarkEmailSent(planRequestID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`
		UPDATE public.tbl_plan_request
		SET email_sent = TRUE, updated_at = now()
		WHERE plan_request_id::text = $1
	`, strings.TrimSpace(planRequestID))
	return err
}

// CountCompanyRequestsSince counts the requests of a company since a time
// (simple protection against repeated sends).
func CountCompanyRequestsSince(companyID string, since time.Time) (int, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return 0, fmt.Errorf("database connection error: %w", err)
	}
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM public.tbl_plan_request
		WHERE company_id = $1 AND created_at >= $2
	`, strings.TrimSpace(companyID), since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count plan requests: %w", err)
	}
	return count, nil
}

// SelectLastPendingRequest returns the newest PENDING request of a company
// (nil when there is none), so the window can say "request sent on ...".
func SelectLastPendingRequest(companyID string) (*PlanRequestS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var r PlanRequestS
	err = db.QueryRow(`
		SELECT plan_request_id::text, requested_plan, status, created_at
		FROM public.tbl_plan_request
		WHERE company_id = $1 AND status = 'PENDING'
		ORDER BY created_at DESC
		LIMIT 1
	`, strings.TrimSpace(companyID)).Scan(&r.PlanRequestID, &r.RequestedPlan, &r.Status, &r.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plan request: %w", err)
	}
	return &r, nil
}

// PlanHistory summarizes the previous requests of one plan by a company
// (used for TRIAL: the platform owner decides if another trial is given).
type PlanHistory struct {
	Requests int        `json:"requests"`
	Approved int        `json:"approved"`
	Last     *time.Time `json:"last,omitempty"`
}

// SelectPlanHistory counts the requests of a plan sent by a company.
func SelectPlanHistory(companyID, planCode string) (*PlanHistory, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var (
		h    PlanHistory
		last sql.NullTime
	)
	err = db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'APPROVED'),
		       MAX(created_at)
		FROM public.tbl_plan_request
		WHERE company_id = $1 AND requested_plan = $2
	`, strings.TrimSpace(companyID), strings.ToUpper(strings.TrimSpace(planCode))).Scan(&h.Requests, &h.Approved, &last)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan history: %w", err)
	}
	if last.Valid {
		h.Last = &last.Time
	}
	return &h, nil
}
