package blpageconfiguration

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/configuration"
)

// defaultShareStoreBaseURL is used when tbl_configuration has no
// SHOWCASE_BASE_URL (same default as the notification emails).
const defaultShareStoreBaseURL = "http://192.168.100.110:3001"

// SharePageData is what the link-preview page shows to WhatsApp, Facebook,
// etc. (title, description, image) and where it sends real visitors.
type SharePageData struct {
	CompanyID   string
	StoreURL    string
	Title       string
	Description string
	ImageURL    string
	Language    string
}

var shareDescriptions = map[string]string{
	"en": "See our products and services and order online.",
	"es": "Mira nuestros productos y servicios y pide en línea.",
}

var shareFallbackTitles = map[string]string{
	"en": "Online store",
	"es": "Tienda en línea",
}

// BSharePageData builds the preview data of a store from its key (slug or
// internal id). Returns ErrPageConfigurationNotFound when the key is unknown.
func BSharePageData(key string) (*SharePageData, error) {
	companyID, found, err := BResolveCompanyKey(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrPageConfigurationNotFound
	}

	data := &SharePageData{CompanyID: companyID, Language: defaultNotificationLanguage}
	storeKey := companyID

	config, err := BSelectPageConfiguration(companyID)
	switch {
	case err == nil:
		data.Title = strings.TrimSpace(config.DisplayName)
		if allowedNotificationLanguages[config.NotificationLanguage] {
			data.Language = config.NotificationLanguage
		}
		if config.HeaderURL != nil {
			data.ImageURL = *config.HeaderURL
		} else if config.LogoURL != nil {
			data.ImageURL = *config.LogoURL
		}
		if config.Slug != nil && *config.Slug != "" {
			storeKey = *config.Slug
		}
	case errors.Is(err, ErrPageConfigurationNotFound):
		// No page configured yet: generic preview.
	default:
		return nil, err
	}

	if data.Title == "" {
		data.Title = shareFallbackTitles[data.Language]
	}
	data.Description = shareDescriptions[data.Language]

	// .env SHOWCASE_BASE_URL wins (2026-10-31), then tbl_configuration.
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("SHOWCASE_BASE_URL")), "/")
	if base == "" {
		base, _ = configuration.GetParameter("SHOWCASE_BASE_URL")
		base = strings.TrimRight(strings.TrimSpace(base), "/")
	}
	if base == "" {
		base = defaultShareStoreBaseURL
	}
	data.StoreURL = base + "/" + url.PathEscape(storeKey)

	return data, nil
}
