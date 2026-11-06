package blcaptcha

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeCloudflare(t *testing.T, status int, body string) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostForm.Get("secret") == "" || r.PostForm.Get("response") == "" {
			t.Errorf("secret and response must be sent")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	previous := turnstileVerifyURL
	turnstileVerifyURL = server.URL
	return func() { turnstileVerifyURL = previous; server.Close() }
}

func TestOffWithoutSecret(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	if err := BVerifyCaptcha(""); err != nil {
		t.Fatalf("without a secret the CAPTCHA must be off, got %v", err)
	}
}

func TestProviderNone(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "x")
	t.Setenv("CAPTCHA_PROVIDER", "none")
	if err := BVerifyCaptcha(""); err != nil {
		t.Fatalf("provider none must be off, got %v", err)
	}
}

func TestRequiredAndValid(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "secret")
	if err := BVerifyCaptcha(""); !errors.Is(err, ErrCaptchaRequired) {
		t.Fatalf("empty token: got %v", err)
	}
	defer fakeCloudflare(t, 200, `{"success":true,"hostname":"store.example.com"}`)()
	if err := BVerifyCaptcha("token"); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	t.Setenv("TURNSTILE_ALLOWED_HOSTNAMES", "store.example.com, www.example.com")
	if err := BVerifyCaptcha("token"); err != nil {
		t.Fatalf("allowed hostname rejected: %v", err)
	}
	t.Setenv("TURNSTILE_ALLOWED_HOSTNAMES", "other.com")
	if err := BVerifyCaptcha("token"); !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("wrong hostname accepted: %v", err)
	}
}

func TestRejected(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "secret")
	defer fakeCloudflare(t, 200, `{"success":false,"error-codes":["timeout-or-duplicate"]}`)()
	if err := BVerifyCaptcha("used-token"); !errors.Is(err, ErrCaptchaFailed) || Code(err) != "CAPTCHA_FAILED" {
		t.Fatalf("got %v", err)
	}
}

func TestUnavailableFailClosedAndOpen(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "secret")
	defer fakeCloudflare(t, 500, `oops`)()
	if err := BVerifyCaptcha("token"); !errors.Is(err, ErrCaptchaUnavailable) {
		t.Fatalf("fail closed expected, got %v", err)
	}
	t.Setenv("CAPTCHA_FAIL_OPEN", "1")
	if err := BVerifyCaptcha("token"); err != nil {
		t.Fatalf("fail open expected, got %v", err)
	}
}
