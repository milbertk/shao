package middlewares

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/mux"
)

// RequestIDHeader is returned with every response. When a company reports a
// problem, this id finds the exact log lines: journalctl -u shao | grep <id>.
const RequestIDHeader = "X-Request-ID"

// RequestLog writes ONE log line per API call and turns a panic into a clean
// 500 answer (2026-11-01). Registered with r.Use in router.NewRouter, so it
// runs after the route is matched and knows the route template.
//
// Logged: request_id, method, route TEMPLATE (e.g.
// /companies/{companyId}/orders, never the real path: some paths carry
// private customer tokens), status, duration_ms, bytes, company_id (from the
// route). Never logged: bodies, query strings, cookies, Authorization.
// Level: 5xx = ERROR, 4xx = WARN, else INFO; a successful /health or / is not
// logged (uptime monitors call it every minute).
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := newRequestID()
		w.Header().Set(RequestIDHeader, requestID)

		route := routeTemplate(r)
		companyID := mux.Vars(r)["companyId"]
		rec := &statusRecorder{ResponseWriter: w}

		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler {
					panic(p) // the server aborts the response on purpose
				}
				slog.Error("panic",
					"request_id", requestID,
					"method", r.Method,
					"route", route,
					"company_id", companyID,
					"error", fmt.Sprint(p),
					"stack", string(debug.Stack()),
				)
				if !rec.wroteHeader {
					rec.Header().Set("Content-Type", "application/json")
					rec.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintf(rec, `{"status":"InternalError","message":"internal server error","requestId":%q}`, requestID)
				} else if rec.status < http.StatusInternalServerError {
					rec.status = http.StatusInternalServerError // the answer was cut
				}
			}
			logRequest(r, requestID, route, companyID, rec, time.Since(start))
		}()

		next.ServeHTTP(rec, r)
	})
}

func logRequest(r *http.Request, requestID, route, companyID string, rec *statusRecorder, elapsed time.Duration) {
	status := rec.statusCode()
	if status < http.StatusBadRequest && (route == "/health" || route == "/") {
		return
	}

	level := slog.LevelInfo
	switch {
	case status >= http.StatusInternalServerError:
		level = slog.LevelError
	case status >= http.StatusBadRequest:
		level = slog.LevelWarn
	}

	attrs := []any{
		"request_id", requestID,
		"method", r.Method,
		"route", route,
		"status", status,
		"duration_ms", elapsed.Milliseconds(),
		"bytes", rec.bytes,
	}
	if companyID != "" {
		attrs = append(attrs, "company_id", companyID)
	}
	slog.Log(r.Context(), level, "request", attrs...)
}

// routeTemplate returns the matched route pattern, never the real path.
func routeTemplate(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil {
			return template
		}
	}
	return "unmatched"
}

// newRequestID returns 12 random hex characters (e.g. "3f9a1c07b2e4").
func newRequestID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// statusRecorder remembers the status code and the size of the answer.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Flush keeps streaming answers working through the recorder.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) statusCode() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}
