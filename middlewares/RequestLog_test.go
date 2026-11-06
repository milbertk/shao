package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// captureLogs sends slog output to a buffer (JSON) during one test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		lines = append(lines, entry)
	}
	return lines
}

func testRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(RequestLog)
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.HandleFunc("/companies/{companyId}/orders", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	r.HandleFunc("/public/manage/{token}", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.HandleFunc("/companies/{companyId}/boom", func(http.ResponseWriter, *http.Request) {
		panic("something nil")
	})
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/service", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("svc")) })
	return r
}

func TestRequestLogOneLinePerRequest(t *testing.T) {
	buf := captureLogs(t)
	rr := httptest.NewRecorder()
	testRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/companies/C123/orders?secret=x", nil))

	id := rr.Header().Get(RequestIDHeader)
	if len(id) != 12 {
		t.Fatalf("request id header = %q", id)
	}
	lines := logLines(t, buf)
	if len(lines) != 1 {
		t.Fatalf("want 1 log line, got %d: %s", len(lines), buf.String())
	}
	got := lines[0]
	if got["request_id"] != id || got["route"] != "/companies/{companyId}/orders" ||
		got["company_id"] != "C123" || got["status"] != float64(400) || got["level"] != "WARN" {
		t.Fatalf("unexpected log line: %v", got)
	}
	if strings.Contains(buf.String(), "secret") {
		t.Fatal("the query string must not be logged")
	}
}

func TestRequestLogNeverLogsTheRealPath(t *testing.T) {
	buf := captureLogs(t)
	testRouter().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/public/manage/PRIVATE-TOKEN-123", nil))
	if strings.Contains(buf.String(), "PRIVATE-TOKEN-123") {
		t.Fatalf("private token logged: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `/public/manage/{token}`) {
		t.Fatalf("route template missing: %s", buf.String())
	}
}

func TestRequestLogPanicReturns500WithRequestID(t *testing.T) {
	buf := captureLogs(t)
	rr := httptest.NewRecorder()
	testRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/companies/C9/boom", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
	id := rr.Header().Get(RequestIDHeader)
	if !strings.Contains(rr.Body.String(), id) {
		t.Fatalf("body must include the request id: %s", rr.Body.String())
	}
	lines := logLines(t, buf)
	if len(lines) != 2 || lines[0]["msg"] != "panic" || lines[1]["status"] != float64(500) || lines[1]["level"] != "ERROR" {
		t.Fatalf("unexpected log lines: %s", buf.String())
	}
	if stack, _ := lines[0]["stack"].(string); !strings.Contains(stack, "RequestLog") {
		t.Fatal("panic line must include the stack")
	}
}

func TestRequestLogSkipsHealthAndCoversSubrouters(t *testing.T) {
	buf := captureLogs(t)
	router := testRouter()
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	if buf.Len() != 0 {
		t.Fatalf("a successful /health must not be logged: %s", buf.String())
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/service", nil))
	lines := logLines(t, buf)
	if len(lines) != 1 || lines[0]["route"] != "/api/service" || lines[0]["bytes"] != float64(3) {
		t.Fatalf("subrouter request not logged correctly: %s", buf.String())
	}
}
