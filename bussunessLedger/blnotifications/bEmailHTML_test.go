package blnotifications

import (
	"strings"
	"testing"

	emailnotifier "github.com/milbertk/shao/shared"
)

func TestRenderEmailHTML(t *testing.T) {
	t.Setenv("ALEPHK_LOGO_URL", "https://cdn.aleph-k.com/logo.png")
	own := companyContext{Name: "Kire <b>", Language: "en", LogoURL: "https://firebasestorage.googleapis.com/x/logo.png", SMTPSource: "COMPANY", SMTP: &emailnotifier.SMTPSettings{}}
	page := renderEmailHTML(own, "Hi Ana <script>,\nSee https://store.aleph-k.com/kire?a=1&b=2")
	for _, want := range []string{`src="https://firebasestorage.googleapis.com/x/logo.png"`, "Hi Ana &lt;script&gt;,<br>", `href="https://store.aleph-k.com/kire?a=1&amp;b=2"`, `alt="Kire &lt;b&gt;"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("missing %q in\n%s", want, page)
		}
	}
	if strings.Contains(page, "<script>") || strings.Contains(page, "AlephK") {
		t.Fatal("own SMTP: no AlephK footer, nothing unescaped")
	}

	def := companyContext{Name: "Kire", Language: "es", LogoURL: "http://192.168.100.110/logo.png", SMTPSource: "DEFAULT"}
	page = renderEmailHTML(def, "Hola")
	if strings.Contains(page, "192.168") || !strings.Contains(page, ">Kire</div>") {
		t.Fatal("non-https logo must fall back to the name")
	}
	if !strings.Contains(page, "Enviado por AlephK en nombre de Kire") || !strings.Contains(page, `src="https://cdn.aleph-k.com/logo.png"`) {
		t.Fatal("AlephK footer missing")
	}
	if onBehalfText(def) == "" || onBehalfText(own) != "" {
		t.Fatal("on behalf text only with the AlephK general email")
	}
}
