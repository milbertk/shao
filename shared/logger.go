package emailnotifier

import (
	"log/slog"
	"os"
	"strings"
)

// SetupLogger sends every log line (slog and the old log.Printf lines) to the
// standard output with a level, in one of two formats:
//
//   - LOG_FORMAT=json: one JSON object per line (production default, when
//     APP_ENV=production). The keys "severity" and "message" are the names
//     Google Cloud Logging understands, so the levels and fields can be
//     filtered there.
//   - LOG_FORMAT=text: readable key=value lines (default on your computer).
//
// systemd keeps the output in the journal: journalctl -u shao.
// Call it once in main, after .env is loaded (2026-11-01).
func SetupLogger() {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	if format == "" {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
			format = "json"
		} else {
			format = "text"
		}
	}
	slog.SetDefault(slog.New(newLogHandler(format)))
}

func newLogHandler(format string) slog.Handler {
	if format == "json" {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: cloudLoggingKeys,
		})
	}
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
}

// cloudLoggingKeys renames level -> severity and msg -> message (JSON only).
func cloudLoggingKeys(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}
