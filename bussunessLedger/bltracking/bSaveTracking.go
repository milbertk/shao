package bltracking

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blcategories "github.com/milbertk/shao/bussunessLedger/categories"
	blproducts "github.com/milbertk/shao/bussunessLedger/products"
	tracking "github.com/milbertk/shao/dataBasesLedger/trackingsteeps"
)

// emptyToNil returns nil when a *string is blank, so optional columns receive
// NULL instead of "".
func emptyToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// validateTarget makes sure the target (category OR product) is set correctly
// and actually exists in the company. Exactly one of them must be provided.
func validateTarget(companyID string, config *tracking.TrackingConfigS) error {
	categoryID := ""
	if config.CategoryID != nil {
		categoryID = strings.TrimSpace(*config.CategoryID)
	}
	productID := ""
	if config.ProductID != nil {
		productID = strings.TrimSpace(*config.ProductID)
	}

	targetType := strings.ToUpper(strings.TrimSpace(config.TargetType))

	switch targetType {
	case "CATEGORY":
		if categoryID == "" {
			return errors.New("categoryId is required for a CATEGORY tracking")
		}
		if productID != "" {
			return errors.New("a CATEGORY tracking cannot have a productId")
		}
		if _, err := blcategories.BSelectCategoryByInternalID(companyID, categoryID); err != nil {
			if errors.Is(err, blcategories.ErrCategoryNotFound) {
				return fmt.Errorf("category %q does not exist in this company", categoryID)
			}
			return fmt.Errorf("failed to validate category: %w", err)
		}
		// Normalize: keep category, clear product.
		config.CategoryID = &categoryID
		config.ProductID = nil

	case "PRODUCT":
		if productID == "" {
			return errors.New("productId is required for a PRODUCT tracking")
		}
		if categoryID != "" {
			return errors.New("a PRODUCT tracking cannot have a categoryId")
		}
		if _, err := blproducts.BSelectProductByInternalID(companyID, productID); err != nil {
			if errors.Is(err, blproducts.ErrProductNotFound) {
				return fmt.Errorf("product %q does not exist in this company", productID)
			}
			return fmt.Errorf("failed to validate product: %w", err)
		}
		// Normalize: keep product, clear category.
		config.ProductID = &productID
		config.CategoryID = nil

	case "GENERAL":
		// A GENERAL tracking is not tied to a category or product; it is
		// chosen during order creation. Clear both targets.
		config.CategoryID = nil
		config.ProductID = nil

	default:
		return errors.New("targetType must be CATEGORY, PRODUCT or GENERAL")
	}

	return nil
}

// BSaveTracking creates or updates a tracking header, then REWRITES its steps
// (delete all + insert the provided list) so step order stays consistent.
//
//   - companyID is authoritative and forced onto every record.
//   - Target: exactly one of category/product, validated to exist in company.
//   - Header: empty/unknown trackingConfigId -> create; otherwise -> update.
//   - Steps: full replace.
//
// Returns the fresh header + ordered steps.
func BSaveTracking(companyID string, input TrackingBundle) (*TrackingBundle, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	input.Config.CompanyID = companyID
	input.Config.TrackingNumber = emptyToNil(input.Config.TrackingNumber)

	// Validate and normalize the target (category or product).
	if err := validateTarget(companyID, &input.Config); err != nil {
		return nil, err
	}

	// ---- Save header (create or update) ----
	configID := strings.TrimSpace(input.Config.TrackingConfigID)

	if configID == "" {
		if err := input.Config.Insert(); err != nil {
			return nil, fmt.Errorf("failed to create tracking config: %w", err)
		}
	} else {
		existing := tracking.TrackingConfigS{
			TrackingConfigID: configID,
			CompanyID:        companyID,
		}
		err := existing.Select()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				input.Config.TrackingConfigID = ""
				if err := input.Config.Insert(); err != nil {
					return nil, fmt.Errorf("failed to create tracking config: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to validate tracking config: %w", err)
			}
		} else {
			if err := input.Config.Update(); err != nil {
				return nil, fmt.Errorf("failed to update tracking config: %w", err)
			}
		}
	}

	configID = input.Config.TrackingConfigID

	// ---- Rewrite steps (full replace) ----
	if _, err := tracking.DeleteStepsByConfig(companyID, configID); err != nil {
		return nil, fmt.Errorf("failed to clear tracking steps: %w", err)
	}

	for index := range input.Steps {
		step := &input.Steps[index]
		step.TrackingConfigID = configID
		step.CompanyID = companyID
		step.DependsOnStepID = emptyToNil(step.DependsOnStepID)

		// Keep a sensible order if the caller did not send one.
		if step.StepOrder < 1 {
			step.StepOrder = index + 1
		}

		if err := step.Insert(); err != nil {
			return nil, fmt.Errorf("failed to insert tracking step: %w", err)
		}
	}

	// ---- Return the fresh state ----
	return BSelectTrackingFull(companyID, configID)
}
