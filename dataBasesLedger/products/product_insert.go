package products

import (
	"fmt"
	"time"

	dbmanager "github.com/milbertk/shao/dataBasesLedger/dbManager"
)

// ProductS represents one row from public.tbl_products.
type ProductS struct {
	ProductInternalID    string    `json:"productInternalId"`
	CompanyID            string    `json:"companyId"`
	ProductCode          string    `json:"productCode"`
	CategoryID           string    `json:"categoryId"`
	Name                 string    `json:"name"`
	Description          *string   `json:"description,omitempty"`
	ProductType          string    `json:"productType"`
	HasPrice             bool      `json:"hasPrice"`
	BasePrice            *float64  `json:"basePrice,omitempty"`
	CurrencySymbol       string    `json:"currencySymbol"` // shown with the price (default "$")
	TaxFree              bool      `json:"taxFree"`
	ShowTaxes            bool      `json:"showTaxes"`
	TaxPercentage        *float64  `json:"taxPercentage,omitempty"`
	ShowOnWeb            bool      `json:"showOnWeb"`
	AvailableForWebSales bool      `json:"availableForWebSales"`
	WebBehavior          string    `json:"webBehavior"`
	WebContactPhone      *string   `json:"webContactPhone,omitempty"`
	WebMessengerLink     *string   `json:"webMessengerLink,omitempty"`
	StepTemplateID       *string   `json:"stepTemplateId,omitempty"`
	Status               string    `json:"status"`
	Comments             *string   `json:"comments,omitempty"`
	ImageURL             *string   `json:"imageUrl,omitempty"`
	ImagePath            *string   `json:"imagePath,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// Insert creates one product for a company.
func (a *ProductS) Insert() error {
	db, err := dbmanager.GetPostgresDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	query := `
		INSERT INTO public.tbl_products (
			company_id,
			product_code,
			category_id,
			name,
			description,
			product_type,
			has_price,
			base_price,
			tax_free,
			show_taxes,
			tax_percentage,
			show_on_web,
			available_for_web_sales,
			web_behavior,
			status,
			comments,
			image_url,
			image_path,
			web_contact_phone,
			web_messenger_link,
			step_template_id,
			currency_symbol
		)
		VALUES (
			$1, $2, $3, $4, $5,
			COALESCE(NULLIF($6, ''), 'SERVICE'),
			$7, $8, $9, $10, $11, $12, $13,
			COALESCE(NULLIF($14, ''), 'MESSENGER_APP'),
			COALESCE(NULLIF($15, ''), 'ACTIVE'),
			$16, $17, $18, $19, $20, $21,
			COALESCE(NULLIF($22, ''), '$')
		)
		RETURNING product_internal_id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		a.CompanyID,
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
		return fmt.Errorf("failed to insert product: %w", err)
	}

	return nil
}
