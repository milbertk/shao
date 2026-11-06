package products

import (
	"database/sql"
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// Update modifies one product identified by primary key (ProductInternalID)
// AND company (CompanyID). updated_at is refreshed by the database trigger.
func (a *ProductS) Update() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	productInternalID := strings.TrimSpace(a.ProductInternalID)
	companyID := strings.TrimSpace(a.CompanyID)
	if productInternalID == "" {
		return fmt.Errorf("productInternalId is required")
	}
	if companyID == "" {
		return fmt.Errorf("companyId is required")
	}

	query := `
		UPDATE public.tbl_products
		SET
			product_code = $3,
			category_id = $4,
			name = $5,
			description = $6,
			product_type = COALESCE(NULLIF($7, ''), 'SERVICE'),
			has_price = $8,
			base_price = $9,
			tax_free = $10,
			show_taxes = $11,
			tax_percentage = $12,
			show_on_web = $13,
			available_for_web_sales = $14,
			web_behavior = COALESCE(NULLIF($15, ''), 'MESSENGER_APP'),
			status = COALESCE(NULLIF($16, ''), 'ACTIVE'),
			comments = $17,
			image_url = $18,
			image_path = $19,
			web_contact_phone = $20,
			web_messenger_link = $21,
			step_template_id = $22,
			currency_symbol = COALESCE(NULLIF($23, ''), '$')
		WHERE product_internal_id = $1
		  AND company_id = $2
		RETURNING product_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		productInternalID,
		companyID,
		a.ProductCode,
		a.CategoryID,
		a.Name,
		a.Description,
		a.ProductType,
		a.HasPrice,
		a.BasePrice,
		a.TaxFree,
		a.ShowTaxes,
		a.TaxPercentage,
		a.ShowOnWeb,
		a.AvailableForWebSales,
		a.WebBehavior,
		a.Status,
		a.Comments,
		a.ImageURL,
		a.ImagePath,
		a.WebContactPhone,
		a.WebMessengerLink,
		a.StepTemplateID,
		a.CurrencySymbol,
	).Scan(
		&a.ProductInternalID,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product %q does not exist", productInternalID)
		}
		return fmt.Errorf("failed to update product: %w", err)
	}

	return nil
}
