package blpageconfiguration

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
)

// ErrSlugTaken is returned when another company already uses the slug.
var ErrSlugTaken = errors.New("SLUG_TAKEN: this store address is already used by another company")

const (
	minSlugLength = 3
	maxSlugLength = 60
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// reservedSlugs are paths used by the apps; a store cannot take them.
var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "app": true, "assets": true, "companies": true,
	"health": true, "help": true, "login": true, "logout": true, "public": true,
	"r": true, "register": true, "s": true, "service": true, "shao": true,
	"showcase": true, "static": true, "store": true, "support": true,
	"privacy": true, "terms": true, "www": true,
}

// accentFree maps common accented letters to plain ones (á -> a, ñ -> n).
var accentFree = map[rune]rune{
	'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a', 'ã': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e',
	'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i',
	'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o', 'õ': 'o',
	'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u',
	'ñ': 'n', 'ç': 'c', 'ý': 'y', 'ÿ': 'y',
}

// NormalizeSlug turns free text into slug form: lowercase, no accents,
// spaces/underscores/dots -> "-", other symbols removed, no double "-".
// "Maderas Finas Ñandú" -> "maderas-finas-nandu".
func NormalizeSlug(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))

	var b strings.Builder
	lastDash := true // avoid a leading "-"
	for _, r := range lowered {
		if plain, ok := accentFree[r]; ok {
			r = plain
		}
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// ValidateSlug checks an already normalized slug.
func ValidateSlug(slug string) error {
	if len(slug) < minSlugLength || len(slug) > maxSlugLength {
		return fmt.Errorf("invalid slug: use between %d and %d characters", minSlugLength, maxSlugLength)
	}
	if !slugPattern.MatchString(slug) {
		return errors.New("invalid slug: use only lowercase letters, numbers and hyphens")
	}
	if reservedSlugs[slug] {
		return fmt.Errorf("invalid slug: %q is reserved, choose another one", slug)
	}
	if LooksLikeCompanyID(slug) {
		return errors.New("invalid slug: it cannot look like an internal id")
	}
	return nil
}

// LooksLikeCompanyID reports whether the value has the internal id (UUID) shape.
func LooksLikeCompanyID(value string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(value))
}

// prepareSlug normalizes, validates and checks uniqueness. Empty -> nil.
func prepareSlug(companyID string, input *string) (*string, error) {
	if input == nil || strings.TrimSpace(*input) == "" {
		return nil, nil
	}
	slug := NormalizeSlug(*input)
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}
	inUse, err := pageconfiguration.SlugInUse(slug, companyID)
	if err != nil {
		return nil, err
	}
	if inUse {
		return nil, ErrSlugTaken
	}
	return &slug, nil
}

// isSlugUniqueViolation detects the unique index error (two saves at once).
func isSlugUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ux_page_configuration_slug")
}

// BResolveCompanyKey turns a store key (slug or internal id) into the
// internal company id. found is false when the key is a slug that no company
// uses. An internal id is returned as is.
func BResolveCompanyKey(key string) (companyID string, found bool, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, nil
	}
	if LooksLikeCompanyID(key) {
		return key, true, nil
	}

	config := pageconfiguration.PageConfigurationS{}
	if err := config.SelectBySlug(NormalizeSlug(key)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return config.CompanyID, true, nil
}
