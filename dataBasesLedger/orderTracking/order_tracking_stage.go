package ordertracking

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// OrderTrackingStageS is one stage of an order's delivery tracking.
type OrderTrackingStageS struct {
	OrderTrackingID string    `json:"orderTrackingId"`
	OrderInternalID string    `json:"orderInternalId"`
	CompanyID       string    `json:"companyId"`
	Stage           string    `json:"stage"`
	Status          string    `json:"status"`
	StageDate       *string   `json:"stageDate,omitempty"` // YYYY-MM-DD
	Comments        *string   `json:"comments,omitempty"`  // legacy single field (optional)
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

const stageColumns = `
	order_tracking_id,
	order_internal_id,
	company_id,
	stage,
	status,
	to_char(stage_date, 'YYYY-MM-DD'),
	comments,
	created_at,
	updated_at
`

func scanStage(row interface{ Scan(dest ...any) error }, s *OrderTrackingStageS) error {
	return row.Scan(
		&s.OrderTrackingID,
		&s.OrderInternalID,
		&s.CompanyID,
		&s.Stage,
		&s.Status,
		&s.StageDate,
		&s.Comments,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
}

// Insert creates one stage.
func (s *OrderTrackingStageS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	query := `
		INSERT INTO public.tbl_order_tracking (
			order_internal_id, company_id, stage, status, stage_date, comments
		)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4,''),'ACTIVE'), $5, $6)
		RETURNING order_tracking_id, created_at, updated_at
	`
	return db.QueryRow(query,
		s.OrderInternalID, s.CompanyID, s.Stage, s.Status, s.StageDate, s.Comments,
	).Scan(&s.OrderTrackingID, &s.CreatedAt, &s.UpdatedAt)
}

// Update modifies one stage by id AND company.
func (s *OrderTrackingStageS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	id := strings.TrimSpace(s.OrderTrackingID)
	company := strings.TrimSpace(s.CompanyID)
	if id == "" || company == "" {
		return fmt.Errorf("orderTrackingId and companyId are required")
	}
	query := `
		UPDATE public.tbl_order_tracking
		SET stage = $3, status = COALESCE(NULLIF($4,''),'ACTIVE'),
		    stage_date = $5, comments = $6
		WHERE order_tracking_id = $1 AND company_id = $2
		RETURNING order_tracking_id, created_at, updated_at
	`
	err = db.QueryRow(query,
		id, company, s.Stage, s.Status, s.StageDate, s.Comments,
	).Scan(&s.OrderTrackingID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("stage %q does not exist", id)
		}
		return fmt.Errorf("failed to update stage: %w", err)
	}
	return nil
}

// SelectStageByID loads one stage.
func SelectStageByID(companyID, id string) (*OrderTrackingStageS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var s OrderTrackingStageS
	q := `SELECT ` + stageColumns + ` FROM public.tbl_order_tracking
		WHERE order_tracking_id = $1 AND company_id = $2`
	if err := scanStage(db.QueryRow(q, strings.TrimSpace(id), strings.TrimSpace(companyID)), &s); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to select stage: %w", err)
	}
	return &s, nil
}

// SelectStagesByOrder returns all stages of an order, oldest first.
func SelectStagesByOrder(companyID, orderInternalID string) ([]OrderTrackingStageS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	q := `SELECT ` + stageColumns + ` FROM public.tbl_order_tracking
		WHERE order_internal_id = $1 AND company_id = $2
		ORDER BY stage_date ASC NULLS FIRST, created_at ASC`
	rows, err := db.Query(q, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select stages: %w", err)
	}
	defer rows.Close()
	list := make([]OrderTrackingStageS, 0)
	for rows.Next() {
		var s OrderTrackingStageS
		if err := scanStage(rows, &s); err != nil {
			return nil, fmt.Errorf("failed to scan stage: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// MostRecentStageStatus returns the status of the most recent stage of an order
// (by stage_date, then created_at). Empty string if the order has no stages.
func MostRecentStageStatus(companyID, orderInternalID string) (string, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return "", fmt.Errorf("database connection error: %w", err)
	}
	q := `
		SELECT status
		FROM public.tbl_order_tracking
		WHERE order_internal_id = $1 AND company_id = $2
		ORDER BY stage_date DESC NULLS LAST, created_at DESC
		LIMIT 1
	`
	var status string
	err = db.QueryRow(q, strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID)).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to read most recent stage status: %w", err)
	}
	return status, nil
}

// UpdateOrderStatus sets the parent order's status (used to sync from the stage).
func UpdateOrderStatus(companyID, orderInternalID, status string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return nil
	}
	_, err = db.Exec(`
		UPDATE public.tbl_order SET status = $3
		WHERE order_internal_id = $1 AND company_id = $2`,
		strings.TrimSpace(orderInternalID), strings.TrimSpace(companyID), status)
	if err != nil {
		return fmt.Errorf("failed to sync order status: %w", err)
	}
	return nil
}
