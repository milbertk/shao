package steptemplate

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// StepTemplateS is the header.
type StepTemplateS struct {
	StepTemplateID string    `json:"stepTemplateId"`
	CompanyID      string    `json:"companyId"`
	Name           string    `json:"name"`
	InternalCode   string    `json:"internalCode"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// StepTemplateStepS is one generic step.
type StepTemplateStepS struct {
	StepTemplateStepID string `json:"stepTemplateStepId"`
	StepTemplateID     string `json:"stepTemplateId"`
	CompanyID          string `json:"companyId"`
	Name               string `json:"name"`
	StepOrder          int    `json:"stepOrder"`
	RequiresPrevious   bool   `json:"requiresPrevious"`
	IsDoneStep         bool   `json:"isDoneStep"`
	// NotifyCustomer (2026-10-31): moving an order item to this step emails the
	// customer only when TRUE (column notify_customer, default FALSE).
	NotifyCustomer bool      `json:"notifyCustomer"`
	CreatedAt      time.Time `json:"createdAt"`
}

const headerColumns = `
	step_template_id, company_id, name, internal_code, active, created_at, updated_at
`

func scanHeader(row interface{ Scan(dest ...any) error }, h *StepTemplateS) error {
	return row.Scan(&h.StepTemplateID, &h.CompanyID, &h.Name, &h.InternalCode, &h.Active, &h.CreatedAt, &h.UpdatedAt)
}

func (h *StepTemplateS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return h.InsertWith(db)
}

// InsertWith inserts the header with db or a transaction (2026-10-30).
func (h *StepTemplateS) InsertWith(db dbmanager.Executor) error {
	return db.QueryRow(`
		INSERT INTO public.tbl_step_template (company_id, name, internal_code, active)
		VALUES ($1, $2, $3, $4)
		RETURNING step_template_id, created_at, updated_at`,
		h.CompanyID, h.Name, h.InternalCode, h.Active,
	).Scan(&h.StepTemplateID, &h.CreatedAt, &h.UpdatedAt)
}

func (h *StepTemplateS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return h.UpdateWith(db)
}

// UpdateWith updates the header with db or a transaction (2026-10-30).
func (h *StepTemplateS) UpdateWith(db dbmanager.Executor) error {
	id := strings.TrimSpace(h.StepTemplateID)
	company := strings.TrimSpace(h.CompanyID)
	if id == "" || company == "" {
		return fmt.Errorf("stepTemplateId and companyId are required")
	}
	err := db.QueryRow(`
		UPDATE public.tbl_step_template
		SET name = $3, internal_code = $4, active = $5
		WHERE step_template_id = $1 AND company_id = $2
		RETURNING step_template_id, created_at, updated_at`,
		id, company, h.Name, h.InternalCode, h.Active,
	).Scan(&h.StepTemplateID, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("step template %q does not exist", id)
		}
		return fmt.Errorf("failed to update step template: %w", err)
	}
	return nil
}

func SelectByID(companyID, id string) (*StepTemplateS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var h StepTemplateS
	q := `SELECT ` + headerColumns + ` FROM public.tbl_step_template
		WHERE step_template_id = $1 AND company_id = $2`
	if err := scanHeader(db.QueryRow(q, strings.TrimSpace(id), strings.TrimSpace(companyID)), &h); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to select step template: %w", err)
	}
	return &h, nil
}

func SelectByCode(companyID, code string) (*StepTemplateS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	var h StepTemplateS
	q := `SELECT ` + headerColumns + ` FROM public.tbl_step_template
		WHERE company_id = $1 AND lower(internal_code) = lower($2)`
	if err := scanHeader(db.QueryRow(q, strings.TrimSpace(companyID), strings.TrimSpace(code)), &h); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to select step template by code: %w", err)
	}
	return &h, nil
}

// Search headers by term (name + internal_code). Only ACTIVE processes
// (used by the product form).
func Search(companyID, term string, limit, offset int) ([]StepTemplateS, error) {
	return SearchFiltered(companyID, term, limit, offset, false)
}

// SearchFiltered is Search that can also return disabled processes
// (includeInactive, used by the process maintenance screen, 2026-10-30).
func SearchFiltered(companyID, term string, limit, offset int, includeInactive bool) ([]StepTemplateS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("companyId is required")
	}

	conditions := []string{"company_id = $1"}
	if !includeInactive {
		conditions = append(conditions, "active = TRUE")
	}
	args := []interface{}{companyID}
	pos := 2
	if t := strings.TrimSpace(term); t != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR internal_code ILIKE $%d)", pos, pos))
		args = append(args, "%"+t+"%")
		pos++
	}
	where := "WHERE " + strings.Join(conditions, " AND ")

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	limitPh := "$" + strconv.Itoa(pos)
	pos++
	offsetPh := "$" + strconv.Itoa(pos)

	q := fmt.Sprintf(`SELECT `+headerColumns+`
		FROM public.tbl_step_template %s
		ORDER BY lower(name) LIMIT %s OFFSET %s`, where, limitPh, offsetPh)
	args = append(args, limit, offset)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search step templates: %w", err)
	}
	defer rows.Close()
	list := make([]StepTemplateS, 0)
	for rows.Next() {
		var h StepTemplateS
		if err := scanHeader(rows, &h); err != nil {
			return nil, fmt.Errorf("failed to scan step template: %w", err)
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

// ---- Steps ----
func (s *StepTemplateStepS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	return s.InsertWith(db)
}

// InsertWith inserts one step with db or a transaction (2026-10-30).
func (s *StepTemplateStepS) InsertWith(db dbmanager.Executor) error {
	if s.StepOrder < 1 {
		s.StepOrder = 1
	}
	return db.QueryRow(`
		INSERT INTO public.tbl_step_template_step
			(step_template_id, company_id, name, step_order, requires_previous, is_done_step, notify_customer)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING step_template_step_id, created_at`,
		s.StepTemplateID, s.CompanyID, s.Name, s.StepOrder, s.RequiresPrevious, s.IsDoneStep, s.NotifyCustomer,
	).Scan(&s.StepTemplateStepID, &s.CreatedAt)
}

func SelectStepsByTemplate(companyID, templateID string) ([]StepTemplateStepS, error) {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}
	return SelectStepsByTemplateWith(db, companyID, templateID)
}

// SelectStepsByTemplateWith reads the steps with db or a transaction (2026-10-30).
func SelectStepsByTemplateWith(db dbmanager.Executor, companyID, templateID string) ([]StepTemplateStepS, error) {
	rows, err := db.Query(`
		SELECT step_template_step_id, step_template_id, company_id, name, step_order, requires_previous, is_done_step,
			notify_customer, created_at
		FROM public.tbl_step_template_step
		WHERE step_template_id = $1 AND company_id = $2
		ORDER BY step_order ASC`,
		strings.TrimSpace(templateID), strings.TrimSpace(companyID))
	if err != nil {
		return nil, fmt.Errorf("failed to select steps: %w", err)
	}
	defer rows.Close()
	list := make([]StepTemplateStepS, 0)
	for rows.Next() {
		var s StepTemplateStepS
		if err := rows.Scan(&s.StepTemplateStepID, &s.StepTemplateID, &s.CompanyID,
			&s.Name, &s.StepOrder, &s.RequiresPrevious, &s.IsDoneStep, &s.NotifyCustomer, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func DeleteStepsByTemplate(companyID, templateID string) error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	_, err = db.Exec(`DELETE FROM public.tbl_step_template_step
		WHERE step_template_id = $1 AND company_id = $2`,
		strings.TrimSpace(templateID), strings.TrimSpace(companyID))
	return err
}

// ---- Steps kept in place (2026-10-30) ----
// A process is saved by updating its existing steps (same ids) instead of
// deleting and re-creating them: orders point to the step ids
// (tbl_order_item.current_step_id, tbl_order_item_step_history) with
// ON DELETE SET NULL, so re-creating them erased the orders' current step.

// UpdateWith changes one existing step (name, order, flags).
func (s *StepTemplateStepS) UpdateWith(db dbmanager.Executor) error {
	result, err := db.Exec(`
		UPDATE public.tbl_step_template_step
		SET name = $4, step_order = $5, requires_previous = $6, is_done_step = $7, notify_customer = $8
		WHERE step_template_step_id = $1 AND step_template_id = $2 AND company_id = $3`,
		strings.TrimSpace(s.StepTemplateStepID), strings.TrimSpace(s.StepTemplateID), strings.TrimSpace(s.CompanyID),
		s.Name, s.StepOrder, s.RequiresPrevious, s.IsDoneStep, s.NotifyCustomer)
	if err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("step %q does not exist", s.StepTemplateStepID)
	}
	return nil
}

// ClearDoneStepWith unflags the "done" step of a template, so the steps can
// be re-ordered without breaking the one-done-step unique index.
func ClearDoneStepWith(db dbmanager.Executor, companyID, templateID string) error {
	_, err := db.Exec(`
		UPDATE public.tbl_step_template_step SET is_done_step = FALSE
		WHERE step_template_id = $1 AND company_id = $2 AND is_done_step = TRUE`,
		strings.TrimSpace(templateID), strings.TrimSpace(companyID))
	if err != nil {
		return fmt.Errorf("failed to clear the done step: %w", err)
	}
	return nil
}

// StepInUseWith tells if any order item uses the step (current step or history).
func StepInUseWith(db dbmanager.Executor, companyID, stepID string) (bool, error) {
	var used bool
	err := db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM public.tbl_order_item WHERE current_step_id = $1)
		    OR EXISTS (SELECT 1 FROM public.tbl_order_item_step_history WHERE step_template_step_id = $1)`,
		strings.TrimSpace(stepID)).Scan(&used)
	if err != nil {
		return false, fmt.Errorf("failed to check if the step is used: %w", err)
	}
	return used, nil
}

// DeleteStepWith deletes one step of a template.
func DeleteStepWith(db dbmanager.Executor, companyID, templateID, stepID string) error {
	_, err := db.Exec(`
		DELETE FROM public.tbl_step_template_step
		WHERE step_template_step_id = $1 AND step_template_id = $2 AND company_id = $3`,
		strings.TrimSpace(stepID), strings.TrimSpace(templateID), strings.TrimSpace(companyID))
	if err != nil {
		return fmt.Errorf("failed to delete step: %w", err)
	}
	return nil
}

// ---- Delete / disable (2026-10-30) ----

// StepTemplateUsage says how much a process is used.
type StepTemplateUsage struct {
	Products   int `json:"products"`   // products with this process
	OrderItems int `json:"orderItems"` // order items on (or that passed) one of its steps
}

// InUse tells if the process cannot be deleted.
func (u StepTemplateUsage) InUse() bool { return u.Products > 0 || u.OrderItems > 0 }

// SelectUsageWith counts the products and order items that use a process.
func SelectUsageWith(db dbmanager.Executor, companyID, templateID string) (StepTemplateUsage, error) {
	var usage StepTemplateUsage
	err := db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM public.tbl_products p
			  WHERE p.company_id = $2 AND p.step_template_id::text = $1),
			(SELECT COUNT(DISTINCT x.item) FROM (
				SELECT oi.order_item_id::text AS item
				  FROM public.tbl_order_item oi
				  JOIN public.tbl_step_template_step s ON s.step_template_step_id = oi.current_step_id
				 WHERE s.step_template_id::text = $1
				UNION
				SELECT h.order_item_id::text AS item
				  FROM public.tbl_order_item_step_history h
				  JOIN public.tbl_step_template_step s ON s.step_template_step_id = h.step_template_step_id
				 WHERE s.step_template_id::text = $1
			) x)`,
		strings.TrimSpace(templateID), strings.TrimSpace(companyID)).Scan(&usage.Products, &usage.OrderItems)
	if err != nil {
		return usage, fmt.Errorf("failed to read process usage: %w", err)
	}
	return usage, nil
}

// SetActiveWith enables or disables a process.
func SetActiveWith(db dbmanager.Executor, companyID, templateID string, active bool) error {
	result, err := db.Exec(`
		UPDATE public.tbl_step_template SET active = $3, updated_at = now()
		WHERE step_template_id::text = $1 AND company_id = $2`,
		strings.TrimSpace(templateID), strings.TrimSpace(companyID), active)
	if err != nil {
		return fmt.Errorf("failed to change the process status: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteTemplateWith deletes a process and its steps.
func DeleteTemplateWith(db dbmanager.Executor, companyID, templateID string) error {
	templateID = strings.TrimSpace(templateID)
	companyID = strings.TrimSpace(companyID)
	if _, err := db.Exec(`DELETE FROM public.tbl_step_template_step
		WHERE step_template_id::text = $1 AND company_id = $2`, templateID, companyID); err != nil {
		return fmt.Errorf("failed to delete the process steps: %w", err)
	}
	result, err := db.Exec(`DELETE FROM public.tbl_step_template
		WHERE step_template_id::text = $1 AND company_id = $2`, templateID, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete the process: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
