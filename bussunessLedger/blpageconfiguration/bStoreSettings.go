package blpageconfiguration

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Store money settings of a company (2026-10-22):
//   - one currency symbol for every price of the store, cart and emails;
//   - optional payment instructions (SINPE Movil, bank account...).

const (
	DefaultCurrencySymbol        = "$"
	maxCurrencySymbolLength      = 5
	maxPaymentInstructionsLength = 1000
)

// StoreSettings is what the public store and the emails need.
type StoreSettings struct {
	CurrencySymbol      string
	PaymentInstructions string
}

// normalizeCurrencySymbol trims the symbol; empty -> "$"; max 5 characters.
func normalizeCurrencySymbol(value string) (string, error) {
	symbol := strings.TrimSpace(value)
	if symbol == "" {
		return DefaultCurrencySymbol, nil
	}
	if utf8.RuneCountInString(symbol) > maxCurrencySymbolLength {
		return "", fmt.Errorf("invalid currencySymbol: maximum length is %d characters", maxCurrencySymbolLength)
	}
	for _, r := range symbol {
		if unicode.IsControl(r) {
			return "", errors.New("invalid currencySymbol: control characters are not allowed")
		}
	}
	return symbol, nil
}

// maxLegalTextLength limits each legal text (terms, privacy) of a company.
const maxLegalTextLength = 20000

// normalizeLegalText trims a legal text (keeps line breaks); empty -> nil, so
// the store does not show the link (2026-10-24).
func normalizeLegalText(field string, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.ReplaceAll(*value, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(text) > maxLegalTextLength {
		return nil, fmt.Errorf("invalid %s: maximum length is %d characters", field, maxLegalTextLength)
	}
	return &text, nil
}

// normalizePaymentInstructions trims the text (keeps line breaks); empty -> nil.
func normalizePaymentInstructions(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.ReplaceAll(*value, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(text) > maxPaymentInstructionsLength {
		return nil, fmt.Errorf("invalid paymentInstructions: maximum length is %d characters", maxPaymentInstructionsLength)
	}
	return &text, nil
}

// BCompanyStoreSettings returns the currency and payment instructions of a
// company. Without a page configuration (or on a read error) it returns "$"
// and no instructions, so the store keeps working.
func BCompanyStoreSettings(companyID string) StoreSettings {
	settings := StoreSettings{CurrencySymbol: DefaultCurrencySymbol}

	config, err := BSelectPageConfiguration(companyID)
	if err != nil {
		if !errors.Is(err, ErrPageConfigurationNotFound) {
			log.Printf("[BCompanyStoreSettings] Could not read page configuration: companyId=%s error=%v", companyID, err)
		}
		return settings
	}

	if symbol := strings.TrimSpace(config.CurrencySymbol); symbol != "" {
		settings.CurrencySymbol = symbol
	}
	if config.PaymentInstructions != nil {
		settings.PaymentInstructions = strings.TrimSpace(*config.PaymentInstructions)
	}
	return settings
}
