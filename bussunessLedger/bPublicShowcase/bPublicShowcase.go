package blpublicshowcase

import (
	"errors"
	"fmt"
	"log"
	"strings"

	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
	"github.com/milbertk/shao/dataBasesLedger/categories"
	"github.com/milbertk/shao/dataBasesLedger/products"
)

// PublicShowcaseItem is a public-safe view of a product for the showcase.
// It deliberately exposes ONLY fields meant to be public (no internal flags,
// no company-private data beyond what a catalog needs).
type PublicShowcaseItem struct {
	ProductInternalID string `json:"productInternalId"`
	ProductCode       string `json:"productCode"`
	CategoryID        string `json:"categoryId"`
	// Real category name for the store chips ("" when the category is
	// inactive or missing). The internal id is never shown to customers.
	CategoryName     string   `json:"categoryName"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	ProductType      string   `json:"productType"`
	HasPrice         bool     `json:"hasPrice"`
	BasePrice        *float64 `json:"basePrice,omitempty"`
	CurrencySymbol   string   `json:"currencySymbol"`
	WebBehavior      string   `json:"webBehavior"`
	WebContactPhone  string   `json:"webContactPhone"`
	WebMessengerLink string   `json:"webMessengerLink"`
	ImageURL         string   `json:"imageUrl"`
	// Kind classifies how the item behaves on the web: "PRODUCT", "SERVICE"
	// or "BOOKING" (a service whose web behavior is BOOKING).
	Kind string `json:"kind"`
}

// classifyKind decides the public "kind" (2026-10-31: simpler rules):
//   - web_behavior BOOKING ("This is an appointment") -> "BOOKING" (Book)
//   - available for web sales                         -> "PRODUCT" (Add to cart)
//   - otherwise                                       -> "SERVICE" (only "Contact us")
func classifyKind(webBehavior string, availableForWebSales bool) string {
	if strings.EqualFold(strings.TrimSpace(webBehavior), "BOOKING") {
		return "BOOKING"
	}
	if availableForWebSales {
		return "PRODUCT"
	}
	return "SERVICE"
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// publicCurrencySymbol returns the product currency symbol, "$" when empty.
func publicCurrencySymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "$"
	}
	return symbol
}

// toPublicItem maps an internal product to the public-safe showcase item.
func toPublicItem(p products.ProductS) PublicShowcaseItem {
	return PublicShowcaseItem{
		ProductInternalID: p.ProductInternalID,
		ProductCode:       p.ProductCode,
		CategoryID:        p.CategoryID,
		Name:              p.Name,
		Description:       derefString(p.Description),
		ProductType:       p.ProductType,
		HasPrice:          p.HasPrice,
		BasePrice:         p.BasePrice,
		CurrencySymbol:    publicCurrencySymbol(p.CurrencySymbol),
		WebBehavior:       p.WebBehavior,
		WebContactPhone:   derefString(p.WebContactPhone),
		WebMessengerLink:  derefString(p.WebMessengerLink),
		ImageURL:          derefString(p.ImageURL),
		Kind:              classifyKind(p.WebBehavior, p.AvailableForWebSales),
	}
}

// BSearchShowcase returns the PUBLIC catalog of one company: web-available,
// active products, optionally filtered. Empty criteria -> all public products.
//
// companyID is REQUIRED (it comes from the public URL, not from a session).
func BSearchShowcase(
	companyID string,
	filters products.PublicProductSearchFilters,
) ([]PublicShowcaseItem, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}

	filters.CompanyID = companyID

	rows, err := products.SearchPublicProducts(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search showcase: %w", err)
	}

	// One currency for the whole store (Company page), so the cart never
	// mixes symbols. Category names for the chips.
	currency := blpageconfiguration.BCompanyStoreSettings(companyID).CurrencySymbol
	categoryNames, err := categories.SelectActiveCategoryNames(companyID)
	if err != nil {
		log.Printf("[BSearchShowcase] Category names not loaded: companyId=%s error=%v", companyID, err)
		categoryNames = map[string]string{}
	}

	items := make([]PublicShowcaseItem, 0, len(rows))
	for _, p := range rows {
		item := toPublicItem(p)
		item.CurrencySymbol = currency
		item.CategoryName = categoryNames[strings.TrimSpace(p.CategoryID)]
		items = append(items, item)
	}

	return items, nil
}
