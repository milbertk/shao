package blnotifications

import (
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
)

// =====================================================================
//  HTML emails with logos (2026-10-31).
//
//  Every notification is sent as text + HTML (email apps that do not show
//  HTML use the text). The HTML version shows:
//    - top:    the company logo (Company page) when it is a public https
//              link; otherwise the company name as text;
//    - body:   the same text as the plain version, links clickable;
//    - footer: only with the AlephK general email: "Sent by AlephK on behalf
//              of <company>" and the AlephK logo (ALEPHK_LOGO_URL in .env,
//              public https link; empty = no logo).
//  Everything coming from the database or customers is HTML-escaped.
// =====================================================================

var emailLinkPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

// publicImageURL returns the URL when it can be shown in an email (https).
func publicImageURL(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "https://") || strings.ContainsAny(value, "\"'<> \r\n") {
		return ""
	}
	return value
}

// alephkLogoURL is the AlephK logo for the footer (ALEPHK_LOGO_URL).
func alephkLogoURL() string {
	return publicImageURL(os.Getenv("ALEPHK_LOGO_URL"))
}

// onBehalfText is the "sent on behalf of" line for the plain text version
// (only with the AlephK general email).
func onBehalfText(c companyContext) string {
	if c.SMTPSource != "DEFAULT" {
		return ""
	}
	return "\n--\n" + fmt.Sprintf(c.texts().onBehalfLine, c.displayName()) + "\n"
}

// bodyToHTML escapes the text, makes links clickable and keeps line breaks.
func bodyToHTML(body string) string {
	escaped := html.EscapeString(strings.ReplaceAll(body, "\r\n", "\n"))
	linked := emailLinkPattern.ReplaceAllStringFunc(escaped, func(link string) string {
		// The link text is already escaped; &amp; stays valid inside href.
		return `<a href="` + link + `" style="color:#0b7d74;word-break:break-all;">` + link + `</a>`
	})
	return strings.ReplaceAll(linked, "\n", "<br>\n")
}

// renderEmailHTML builds the HTML version of one notification.
func renderEmailHTML(c companyContext, body string) string {
	name := html.EscapeString(c.displayName())
	lang := "en"
	if c.Language == "es" {
		lang = "es"
	}

	var top string
	if logo := publicImageURL(c.LogoURL); logo != "" {
		top = `<img src="` + html.EscapeString(logo) + `" alt="` + name + `" style="max-height:64px;max-width:220px;height:auto;border:0;display:block;margin:0 auto;">`
	} else {
		top = `<div style="font-size:20px;font-weight:700;color:#14243b;text-align:center;">` + name + `</div>`
	}

	footer := ""
	if c.SMTPSource == "DEFAULT" {
		t := c.texts()
		footer = `<tr><td style="padding:16px 24px;border-top:1px solid #e3e7ee;color:#6b7686;font-size:12px;line-height:1.5;text-align:center;">` +
			html.EscapeString(fmt.Sprintf(t.onBehalfLine, c.displayName()))
		if logo := alephkLogoURL(); logo != "" {
			footer += `<br><img src="` + html.EscapeString(logo) + `" alt="AlephK" style="height:22px;width:auto;border:0;margin-top:10px;opacity:.85;">`
		} else {
			footer += `<br><span style="color:#14243b;font-weight:600;">` + html.EscapeString(t.poweredByLine) + `</span>`
		}
		footer += `</td></tr>`
	}

	return `<!doctype html>
<html lang="` + lang + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + name + `</title></head>
<body style="margin:0;padding:0;background:#f3f5f9;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f3f5f9;padding:24px 12px;">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;background:#ffffff;border-radius:10px;border:1px solid #e3e7ee;font-family:Arial,Helvetica,sans-serif;">
<tr><td style="padding:20px 24px;border-bottom:3px solid #00b8a9;">` + top + `</td></tr>
<tr><td style="padding:20px 24px;color:#1f2a37;font-size:15px;line-height:1.6;">` + bodyToHTML(body) + `</td></tr>
` + footer + `
</table>
</td></tr>
</table>
</body></html>`
}
