package products

import (
	"fmt"
	"strings"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

const productSelectColumns = `
	product_internal_id,
	company_id,
	product_code,
	category_id,
	name,
	description,
	product_type,
	has_price,
	base_price,
	currency_symbol,
	tax_free,
	show_taxes,
	tax_percentage,
	show_on_web,
	available_for_web_sales,
	web_behavior,
	web_contact_phone,
	web_messenger_link,
	step_template_id,
	status,
	comments,
	image_url,
	image_path,
	created_at,
	updated_at
`

// scanProduct scans one row (in the productSelectColumns order) into p.
func scanProduct(row interface {
	Scan(dest ...any) error
}, p *ProductS) error {
	return row.Scan(
		&p.ProductInternalID,
		&p.CompanyID,
		&p.ProductCode,
		&p.CategoryID,
		&p.Name,
		&p.Description,
		&p.ProductType,
		&p.HasPrice,
		&p.BasePrice,
		&p.CurrencySymbol,
		&p.TaxFree,
		&p.ShowTaxes,
		&p.TaxPercentage,
		&p.ShowOnWeb,
		&p.AvailableForWebSales,
		&p.WebBehavior,
		&p.WebContactPhone,
		&p.WebMessengerLink,
		&p.StepTemplateID,
		&p.Status,
		&p.Comments,
		&p.ImageURL,
		&p.ImagePath,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
}

// Select loads one product by primary key (ProductInternalID), scoped to its
// company. Returns sql.ErrNoRows (wrapped) when not found in that company.
func (a *ProductS) Select() error {
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
		SELECT ` + productSelectColumns + `
		FROM public.tbl_products
		WHERE product_internal_id = $1
		  AND company_id = $2
	`

	if err := scanProduct(db.QueryRow(query, productInternalID, companyID), a); err != nil {
		return fmt.Errorf("failed to select product: %w", err)
	}

	return nil
}
