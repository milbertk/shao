package emailnotifier

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// TestCloudLoggingKeys checks the JSON keys Google Cloud Logging reads.
func TestCloudLoggingKeys(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{ReplaceAttr: cloudLoggingKeys}))
	logger.Warn("hello", "request_id", "abc")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["severity"] != "WARN" || entry["message"] != "hello" || entry["request_id"] != "abc" {
		t.Fatalf("unexpected keys: %v", entry)
	}
}
