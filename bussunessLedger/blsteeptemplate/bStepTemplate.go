package blsteptemplate

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
	"github.com/milbertk/shao/dataBasesLedger/steptemplate"
)

var ErrStepTemplateNotFound = errors.New("step template not found")

// StepTemplateBundle groups the header with its steps.
type StepTemplateBundle struct {
	Template steptemplate.StepTemplateS       `json:"template"`
	Steps    []steptemplate.StepTemplateStepS `json:"steps"`
	// Usage (2026-10-30): products / order items using it; the screen shows
	// Delete when it is not used, Disable / Enable otherwise. Output only.
	Usage *steptemplate.StepTemplateUsage `json:"usage,omitempty"`
}

// ErrStepTemplateInUse: a used process cannot be deleted (HTTP 409).
var ErrStepTemplateInUse = errors.New("The process is used by products or orders, so it cannot be deleted. Disable it instead: it will not be offered for new products.")

func BSelectStepTemplateFull(companyID, id string) (*StepTemplateBundle, error) {
	companyID = strings.TrimSpace(companyID)
	id = strings.TrimSpace(id)
	if companyID == "" || id == "" {
		return nil, errors.New("companyId and id are required")
	}
	header, err := steptemplate.SelectByID(companyID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStepTemplateNotFound
		}
		return nil, fmt.Errorf("failed to select step template: %w", err)
	}
	steps, err := steptemplate.SelectStepsByTemplate(companyID, id)
	if err != nil {
		return nil, err
	}
	bundle := &StepTemplateBundle{Template: *header, Steps: steps}
	if db, err := dbmanager.GetPostgresDB(); err == nil {
		if usage, err := steptemplate.SelectUsageWith(db, companyID, id); err == nil {
			bundle.Usage = &usage
		}
	}
	return bundle, nil
}

func BSearchStepTemplates(companyID, term string, limit, offset int) ([]steptemplate.StepTemplateS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	return steptemplate.Search(companyID, term, limit, offset)
}

// BSearchStepTemplatesFiltered can also list disabled processes (maintenance
// screen, 2026-10-30).
func BSearchStepTemplatesFiltered(companyID, term string, limit, offset int, includeInactive bool) ([]steptemplate.StepTemplateS, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	return steptemplate.SearchFiltered(companyID, term, limit, offset, includeInactive)
}

// BDeleteStepTemplate deletes a process that no product or order uses
// (header + steps in one transaction). A used one -> ErrStepTemplateInUse.
func BDeleteStepTemplate(companyID, id string) error {
	companyID = strings.TrimSpace(companyID)
	id = strings.TrimSpace(id)
	if companyID == "" || id == "" {
		return errors.New("companyId and stepTemplateId are required")
	}
	return dbmanager.WithTx(func(tx *sql.Tx) error {
		usage, err := steptemplate.SelectUsageWith(tx, companyID, id)
		if err != nil {
			return err
		}
		if usage.InUse() {
			return ErrStepTemplateInUse
		}
		if err := steptemplate.DeleteTemplateWith(tx, companyID, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrStepTemplateNotFound
			}
			return err
		}
		return nil
	})
}

// BSetStepTemplateActive enables or disables a process. A disabled process is
// not offered in the product form; products and orders that use it keep
// working with it.
func BSetStepTemplateActive(companyID, id string, active bool) error {
	companyID = strings.TrimSpace(companyID)
	id = strings.TrimSpace(id)
	if companyID == "" || id == "" {
		return errors.New("companyId and stepTemplateId are required")
	}
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}
	if err := steptemplate.SetActiveWith(db, companyID, id, active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStepTemplateNotFound
		}
		return err
	}
	return nil
}

// ErrStepInUse: a step that orders already use cannot be removed (HTTP 409).
var ErrStepInUse = errors.New("step in use")

// StepInUseError names the step that cannot be removed.
type StepInUseError struct {
	Name string
}

func (e *StepInUseError) Error() string {
	return fmt.Sprintf("The step %q is used by existing orders, so it cannot be removed. Keep it, or rename it instead of deleting it.", e.Name)
}

func (e *StepInUseError) Unwrap() error { return ErrStepInUse }

// cleanSteps keeps the steps with a name, numbers them 1..n, makes the first
// one not require a previous step and keeps ONE "done" step: the flagged one
// nearest to the end, or the last step when none is flagged (pure, tested).
func cleanSteps(input []steptemplate.StepTemplateStepS) []steptemplate.StepTemplateStepS {
	steps := make([]steptemplate.StepTemplateStepS, 0, len(input))
	for _, st := range input {
		st.Name = strings.TrimSpace(st.Name)
		if st.Name == "" {
			continue
		}
		steps = append(steps, st)
	}
	doneIndex := -1
	for i := range steps {
		if steps[i].IsDoneStep {
			doneIndex = i
		}
	}
	if doneIndex == -1 && len(steps) > 0 {
		doneIndex = len(steps) - 1
	}
	for i := range steps {
		steps[i].StepOrder = i + 1
		steps[i].IsDoneStep = i == doneIndex
		if i == 0 {
			steps[i].RequiresPrevious = false
		}
	}
	return steps
}

// stepMergePlan says how the saved steps change (pure, tested): the first
// len(existing) new steps reuse the existing rows by position (same ids, so
// orders keep their current step), the rest are inserted and the existing
// rows left over are removed.
type stepMergePlan struct {
	Updates []steptemplate.StepTemplateStepS
	Inserts []steptemplate.StepTemplateStepS
	Removes []steptemplate.StepTemplateStepS
}

func planStepMerge(existing, incoming []steptemplate.StepTemplateStepS) stepMergePlan {
	var plan stepMergePlan
	for i, st := range incoming {
		if i < len(existing) {
			st.StepTemplateStepID = existing[i].StepTemplateStepID
			plan.Updates = append(plan.Updates, st)
		} else {
			st.StepTemplateStepID = ""
			plan.Inserts = append(plan.Inserts, st)
		}
	}
	if len(existing) > len(incoming) {
		plan.Removes = append(plan.Removes, existing[len(incoming):]...)
	}
	return plan
}

// BSaveStepTemplate creates or updates the header (by id, or by internal_code
// when there is no id) and its steps.
//
// 2026-10-30: (1) a process saved from the form is always ACTIVE (before, an
// update wrote active = false because the form does not send it, so the
// process disappeared from the search and the product list); (2) everything
// runs in ONE transaction: on any error nothing is changed; (3) existing
// steps are updated in place (same ids) instead of delete + re-insert, and a
// step used by orders is never deleted.
func BSaveStepTemplate(companyID string, input StepTemplateBundle) (*StepTemplateBundle, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	input.Template.CompanyID = companyID
	input.Template.Name = strings.TrimSpace(input.Template.Name)
	input.Template.InternalCode = strings.TrimSpace(input.Template.InternalCode)
	input.Template.Active = true // new processes; an update keeps its state (below)
	if input.Template.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.Template.InternalCode == "" {
		return nil, errors.New("internalCode is required")
	}
	steps := cleanSteps(input.Steps)
	if len(steps) == 0 {
		return nil, errors.New("at least one step is required")
	}

	id := strings.TrimSpace(input.Template.StepTemplateID)

	// If no id but the code already exists, update that one.
	if id == "" {
		existing, err := steptemplate.SelectByCode(companyID, input.Template.InternalCode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check code: %w", err)
		}
		if existing != nil {
			id = existing.StepTemplateID
		}
	}
	// An id that does not exist (deleted meanwhile) creates a new process.
	if id != "" {
		current, err := steptemplate.SelectByID(companyID, id)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("failed to validate step template: %w", err)
			}
			id = ""
		} else {
			// Saving keeps it enabled / disabled (2026-10-30): only the
			// Enable / Disable buttons change it.
			input.Template.Active = current.Active
		}
	}
	input.Template.StepTemplateID = id

	err := dbmanager.WithTx(func(tx *sql.Tx) error {
		if id == "" {
			if err := input.Template.InsertWith(tx); err != nil {
				return fmt.Errorf("failed to create step template: %w", err)
			}
		} else if err := input.Template.UpdateWith(tx); err != nil {
			return fmt.Errorf("failed to update step template: %w", err)
		}
		templateID := input.Template.StepTemplateID

		existing, err := steptemplate.SelectStepsByTemplateWith(tx, companyID, templateID)
		if err != nil {
			return err
		}
		for i := range steps {
			steps[i].StepTemplateID = templateID
			steps[i].CompanyID = companyID
		}
		plan := planStepMerge(existing, steps)

		// Removed steps first (only when no order uses them).
		for _, old := range plan.Removes {
			used, err := steptemplate.StepInUseWith(tx, companyID, old.StepTemplateStepID)
			if err != nil {
				return err
			}
			if used {
				return &StepInUseError{Name: old.Name}
			}
			if err := steptemplate.DeleteStepWith(tx, companyID, templateID, old.StepTemplateStepID); err != nil {
				return err
			}
		}
		// The "done" flag may move to another step: clear it first
		// (unique index step_template_one_done_uq).
		if err := steptemplate.ClearDoneStepWith(tx, companyID, templateID); err != nil {
			return err
		}
		for i := range plan.Updates {
			if err := plan.Updates[i].UpdateWith(tx); err != nil {
				return err
			}
		}
		for i := range plan.Inserts {
			if err := plan.Inserts[i].InsertWith(tx); err != nil {
				return fmt.Errorf("failed to insert step: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return BSelectStepTemplateFull(companyID, input.Template.StepTemplateID)
}
