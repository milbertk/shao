package publicshowcaseinshao

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/milbertk/shao/HandlerLedgers/handlertracking"
	blpageconfiguration "github.com/milbertk/shao/bussunessLedger/blpageconfiguration"
)

// sharePageTemplate is the small page behind the "share" link. WhatsApp,
// Facebook, etc. read the og:* tags for the preview (they do not run the
// store's JavaScript); real visitors are sent to the store right away.
var sharePageTemplate = template.Must(template.New("share").Parse(`<!doctype html>
<html lang="{{.Language}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<meta name="description" content="{{.Description}}">
<meta property="og:type" content="website">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:url" content="{{.ShareURL}}">
{{if .ImageURL}}<meta property="og:image" content="{{.ImageURL}}">
<meta name="twitter:image" content="{{.ImageURL}}">{{end}}
<meta name="twitter:card" content="{{if .ImageURL}}summary_large_image{{else}}summary{{end}}">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Description}}">
<meta http-equiv="refresh" content="0; url={{.StoreURL}}">
<style>body{font-family:system-ui,sans-serif;display:grid;place-items:center;min-height:90vh;color:#334155}a{color:#0f766e}</style>
</head>
<body>
<p><a href="{{.StoreURL}}">{{.Title}}</a></p>
<script>window.location.replace({{.StoreURL}});</script>
</body>
</html>
`))

type sharePageView struct {
	Language    string
	Title       string
	Description string
	ImageURL    string
	StoreURL    string
	ShareURL    string
}

// HPublicSharePage serves the link-preview page of a store.
//
//	GET /s/{companyKey}     (companyKey = slug or internal id)
//
// PUBLIC and read-only. Shows title (company page name), description and
// image (header or logo) for link previews, then opens the store.
func HPublicSharePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Use GET", http.StatusMethodNotAllowed)
		return
	}

	key := strings.TrimSpace(mux.Vars(r)["companyKey"])
	data, err := blpageconfiguration.BSharePageData(key)
	if err != nil {
		if errors.Is(err, blpageconfiguration.ErrPageConfigurationNotFound) {
			handlertracking.RecordPublic(r, "PublicSharePageNotFound", "", handlertracking.Details{"key": key})
			http.Error(w, "Store not found", http.StatusNotFound)
			return
		}
		log.Printf("[HPublicSharePage] Business operation failed: key=%s error=%v", key, err)
		http.Error(w, "An internal error occurred", http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	shareURL := scheme + "://" + r.Host + r.URL.EscapedPath()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Best-effort action tracking (never blocks the response).
	handlertracking.RecordPublic(r, "PublicSharePageView", data.CompanyID, handlertracking.Details{
		"key":       key,
		"userAgent": r.UserAgent(),
	})

	if err := sharePageTemplate.Execute(w, sharePageView{
		Language:    data.Language,
		Title:       data.Title,
		Description: data.Description,
		ImageURL:    data.ImageURL,
		StoreURL:    data.StoreURL,
		ShareURL:    shareURL,
	}); err != nil {
		log.Printf("[HPublicSharePage] Template error: %v", err)
	}
}
