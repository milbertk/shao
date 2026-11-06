package emailnotifier

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/milbertk/shao/dataBasesLedger/configuration"
)

// EmailMessage is a text email, optionally with an HTML version (2026-10-31:
// HTMLBody; email apps show the HTML and fall back to the text). If To is
// empty, the default recipient from the configuration table (SMTP_TO_DEFAULT)
// is used by Send.
type EmailMessage struct {
	To       string
	ReplyTo  string
	Subject  string
	Body     string
	HTMLBody string
	// Bcc (2026-10-31): blind copies (e.g. the company customer service).
	// They receive the same email; their address is not in the headers.
	Bcc []string
}

// SMTPSettings is the mail server used to send one email. Companies configure
// their own server (tbl_company_smtp); the password arrives here decrypted.
type SMTPSettings struct {
	Host      string
	Port      int
	Security  string // STARTTLS | SSL
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

const smtpTimeout = 20 * time.Second

// DefaultRecipient reads the default notification recipient from the config
// table (SMTP_TO_DEFAULT). Returns "" when it is not configured.
func DefaultRecipient() string {
	to, _ := configuration.GetParameter("SMTP_TO_DEFAULT")
	return strings.TrimSpace(to)
}

// Send sends an email using the PLATFORM SMTP settings from tbl_configuration
// (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM). Kept for
// platform emails. Company notifications use SendWithSettings with the
// company's own SMTP. Without a recipient or SMTP configuration nothing is
// sent (no personal fallback address).
func Send(msg EmailMessage) error {
	cfg, err := configuration.GetParameters(
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD",
		"SMTP_FROM", "SMTP_TO_DEFAULT",
	)
	if err != nil {
		log.Printf("[email] Could not read SMTP configuration: %v", err)
		cfg = map[string]string{}
	}

	to := strings.TrimSpace(msg.To)
	if to == "" {
		to = strings.TrimSpace(cfg["SMTP_TO_DEFAULT"])
	}
	if to == "" {
		log.Printf("[email] No recipient; email not sent (subject: %s)", msg.Subject)
		return nil
	}

	host := strings.TrimSpace(cfg["SMTP_HOST"])
	user := strings.TrimSpace(cfg["SMTP_USER"])
	if host == "" || user == "" {
		log.Printf("[email] Platform SMTP not configured; email to %s not sent (subject: %s)", MaskEmail(to), msg.Subject)
		return nil
	}

	port, _ := strconv.Atoi(strings.TrimSpace(cfg["SMTP_PORT"]))
	if port == 0 {
		port = 587
	}
	from := strings.TrimSpace(cfg["SMTP_FROM"])
	if from == "" {
		from = user
	}
	security := "STARTTLS"
	if port == 465 {
		security = "SSL"
	}

	msg.To = to
	return SendWithSettings(SMTPSettings{
		Host:      host,
		Port:      port,
		Security:  security,
		Username:  user,
		Password:  cfg["SMTP_PASSWORD"],
		FromEmail: from,
	}, msg)
}

// SendWithSettings sends one plain-text UTF-8 email with the given SMTP
// server. Supports STARTTLS (usually port 587) and SSL/TLS (usually 465).
// TLS is always required, so the password never travels in clear text.
func SendWithSettings(settings SMTPSettings, msg EmailMessage) error {
	host := strings.TrimSpace(settings.Host)
	if host == "" || settings.Port <= 0 {
		return errors.New("SMTP host and port are required")
	}

	from, err := cleanAddress(settings.FromEmail)
	if err != nil {
		return fmt.Errorf("invalid sender email: %w", err)
	}
	to, err := cleanAddress(msg.To)
	if err != nil {
		return fmt.Errorf("invalid recipient email: %w", err)
	}
	replyTo := ""
	if strings.TrimSpace(msg.ReplyTo) != "" {
		if replyTo, err = cleanAddress(msg.ReplyTo); err != nil {
			replyTo = ""
		}
	}

	addr := net.JoinHostPort(host, strconv.Itoa(settings.Port))
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: smtpTimeout}

	var client *smtp.Client
	if strings.EqualFold(settings.Security, "SSL") {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("could not connect to %s (SSL): %w", addr, err)
		}
		_ = conn.SetDeadline(time.Now().Add(2 * smtpTimeout))
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP handshake failed: %w", err)
		}
	} else {
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("could not connect to %s: %w", addr, err)
		}
		_ = conn.SetDeadline(time.Now().Add(2 * smtpTimeout))
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("SMTP handshake failed: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); !ok {
			client.Close()
			return errors.New("the server does not support STARTTLS; use SSL (port 465)")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}
	defer client.Close()

	if strings.TrimSpace(settings.Username) != "" {
		auth := smtp.PlainAuth("", settings.Username, settings.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP login failed (check user and password): %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("sender rejected: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("recipient rejected: %w", err)
	}
	for _, blind := range cleanBcc(to, msg.Bcc) {
		if err := client.Rcpt(blind); err != nil {
			// A wrong blind copy never stops the customer email.
			log.Printf("[email] blind copy %s rejected: %v", MaskEmail(blind), err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	message := buildMIME(from, settings.FromName, to, replyTo, msg.Subject, msg.Body)
	if strings.TrimSpace(msg.HTMLBody) != "" {
		message = buildMIMEAlternative(from, settings.FromName, to, replyTo, msg.Subject, msg.Body, msg.HTMLBody)
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write the email: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("the server did not accept the email: %w", err)
	}
	_ = client.Quit()

	log.Printf("[email] Sent to %s via %s (subject: %s)", MaskEmail(to), host, msg.Subject)
	return nil
}

// MaskEmail hides most of an address for server logs (2026-10-25, security
// S11): "maria.perez@gmail.com" -> "m***@gmail.com". Personal data should not
// be written in full to the logs.
func MaskEmail(value string) string {
	value = strings.TrimSpace(value)
	at := strings.LastIndex(value, "@")
	if at <= 0 {
		if value == "" {
			return ""
		}
		return "***"
	}
	return value[:1] + "***" + value[at:]
}

// ValidEmail reports whether value is one plain email address.
func ValidEmail(value string) bool {
	_, err := cleanAddress(value)
	return err == nil
}

// cleanAddress validates one email address and rejects header injection.
func cleanAddress(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty address")
	}
	if strings.ContainsAny(value, "\r\n<>,;") {
		return "", errors.New("invalid characters")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return "", errors.New("invalid address")
	}
	return parsed.Address, nil
}

func headerSafe(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.ReplaceAll(value, "\n", " ")
}

func buildMIME(from, fromName, to, replyTo, subject, body string) string {
	fromHeader := from
	if name := strings.TrimSpace(headerSafe(fromName)); name != "" {
		fromHeader = (&mail.Address{Name: name, Address: from}).String()
	}

	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domain := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 {
		domain = from[at+1:]
	}

	var b strings.Builder
	b.WriteString("From: " + fromHeader + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	if replyTo != "" {
		b.WriteString("Reply-To: " + replyTo + "\r\n")
	}
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", headerSafe(subject)) + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("Message-ID: <" + hex.EncodeToString(id) + "@" + domain + ">\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	return b.String()
}

// buildMIMEAlternative builds a text + HTML email (multipart/alternative,
// 2026-10-31). Both parts are quoted-printable so long HTML lines are safe.
func buildMIMEAlternative(from, fromName, to, replyTo, subject, textBody, htmlBody string) string {
	head := buildMIME(from, fromName, to, replyTo, subject, "")
	// Keep the common headers, replace the single-part content headers.
	cut := strings.Index(head, "Content-Type: text/plain")
	if cut < 0 {
		cut = len(head)
	}
	boundaryBytes := make([]byte, 12)
	_, _ = rand.Read(boundaryBytes)
	boundary := "ak-" + hex.EncodeToString(boundaryBytes)

	var b strings.Builder
	b.WriteString(head[:cut])
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	writePart := func(contentType, content string) {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + contentType + "; charset=\"utf-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		var qp strings.Builder
		w := quotedprintable.NewWriter(&qp)
		_, _ = w.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\n", "\r\n")))
		_ = w.Close()
		b.WriteString(qp.String() + "\r\n")
	}
	writePart("text/plain", textBody)
	writePart("text/html", htmlBody)
	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}

// cleanBcc returns the valid blind-copy addresses, without duplicates and
// without the main recipient (2026-10-31).
func cleanBcc(to string, bcc []string) []string {
	out := []string{}
	seen := map[string]bool{strings.ToLower(to): true}
	for _, item := range bcc {
		address, err := cleanAddress(item)
		if err != nil || seen[strings.ToLower(address)] {
			continue
		}
		seen[strings.ToLower(address)] = true
		out = append(out, address)
	}
	return out
}
