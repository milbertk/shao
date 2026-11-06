package router

import (
	"bufio"
	"log"
	"os"
	"strings"

	emailnotifier "github.com/milbertk/shao/shared"
)

// CORS by environment (2026-10-25, security S9).
//
//	APP_ENV=production -> ONLY the origins in CORS_ALLOWED_ORIGINS and in the
//	                      origins file (CORS_ORIGINS_FILE, default
//	                      ./cors_allowed_origins.txt, one origin per line,
//	                      # = comment). ALLOW_ANY_ORIGIN is IGNORED and the
//	                      local/LAN origins are NOT allowed.
//	APP_ENV=test       -> any origin is allowed (testing only).
//	APP_ENV not set    -> same as before: local/LAN list + CORS_ALLOWED_ORIGINS,
//	                      and ALLOW_ANY_ORIGIN=1 still works.
//
// Domains (2026-10-31): ALLOWED_DOMAINS=aleph-k.com,aleph-k.online (in .env)
// allows each domain AND all its subdomains, with http and https, in every
// environment. A line without "http(s)://" in the origins file or in
// CORS_ALLOWED_ORIGINS (e.g. "aleph-k.com" or "*.aleph-k.com") is a domain too.

const defaultCORSOriginsFile = "./cors_allowed_origins.txt"

// corsDomains: domains (+ subdomains) allowed by ALLOWED_DOMAINS and by the
// domain lines of the origins file / CORS_ALLOWED_ORIGINS.
var corsDomains []string

// originAllowedByDomain tells if the origin is on one of corsDomains.
func originAllowedByDomain(origin string) bool {
	return emailnotifier.OriginAllowedByDomains(origin, corsDomains)
}

func appEnv() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
}

func isProductionEnv() bool {
	env := appEnv()
	return env == "production" || env == "prod"
}

// corsAllowAnyOrigin reports whether every origin is accepted.
func corsAllowAnyOrigin() bool {
	if isProductionEnv() {
		return false
	}
	if appEnv() == "test" {
		return true
	}
	return os.Getenv("ALLOW_ANY_ORIGIN") == "1"
}

// loadCORSOrigins builds the allowed list for the current environment.
func loadCORSOrigins(localOrigins map[string]bool) map[string]bool {
	result := map[string]bool{}
	corsDomains = emailnotifier.AllowedDomains()
	if !isProductionEnv() {
		for origin := range localOrigins {
			result[origin] = true
		}
	}
	for _, origin := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		addCORSOrigin(result, origin)
	}
	file := strings.TrimSpace(os.Getenv("CORS_ORIGINS_FILE"))
	if file == "" {
		file = defaultCORSOriginsFile
	}
	if f, err := os.Open(file); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			addCORSOrigin(result, line)
		}
		f.Close()
	}

	switch {
	case isProductionEnv():
		if os.Getenv("ALLOW_ANY_ORIGIN") == "1" {
			log.Println("[CORS] APP_ENV=production: ALLOW_ANY_ORIGIN is ignored")
		}
		if len(result) == 0 && len(corsDomains) == 0 {
			log.Println("[CORS] WARNING: APP_ENV=production and no allowed origins: set ALLOWED_DOMAINS, CORS_ALLOWED_ORIGINS or " + file)
		}
		log.Printf("[CORS] production: %d allowed origin(s)", len(result))
	case appEnv() == "test":
		log.Println("[CORS] APP_ENV=test: ANY origin is allowed (never use this in production)")
	default:
		log.Println("[CORS] APP_ENV not set: local origins + CORS_ALLOWED_ORIGINS (set APP_ENV=production on the server)")
	}
	if len(corsDomains) > 0 {
		log.Printf("[CORS] allowed domains (with all subdomains, http and https): %s", strings.Join(corsDomains, ", "))
	}
	return result
}

// addCORSOrigin adds one exact origin ("https://app.example.com"), or a
// domain when the entry has no scheme ("example.com", "*.example.com").
func addCORSOrigin(list map[string]bool, origin string) {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return
	}
	if !strings.Contains(origin, "://") {
		if domain := emailnotifier.NormalizeDomain(origin); strings.Contains(domain, ".") {
			for _, existing := range corsDomains {
				if existing == domain {
					return
				}
			}
			corsDomains = append(corsDomains, domain)
		}
		return
	}
	list[origin] = true
}
