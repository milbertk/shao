package blplan

import (
	"reflect"
	"testing"

	"github.com/milbertk/shao/dataBasesLedger/pageconfiguration"
)

func textPtr(value string) *string { return &value }

func TestMissingBasicCompanyFields(t *testing.T) {
	cases := []struct {
		name   string
		config *pageconfiguration.PageConfigurationS
		want   []string
	}{
		{"nothing saved", nil, []string{BasicFieldName, BasicFieldLogo, BasicFieldHeader}},
		{"only name", &pageconfiguration.PageConfigurationS{DisplayName: "Kire"}, []string{BasicFieldLogo, BasicFieldHeader}},
		{"blank values", &pageconfiguration.PageConfigurationS{DisplayName: "  ", LogoURL: textPtr(" "), HeaderURL: textPtr("")},
			[]string{BasicFieldName, BasicFieldLogo, BasicFieldHeader}},
		{"complete", &pageconfiguration.PageConfigurationS{DisplayName: "Kire", LogoURL: textPtr("https://x/logo.png"), HeaderURL: textPtr("https://x/h.png")},
			[]string{}},
	}
	for _, c := range cases {
		got := missingBasicCompanyFields(c.config)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBasicCompanyError(t *testing.T) {
	err := error(&BasicCompanyError{Code: CodeBasicCompanyRequired, Missing: []string{BasicFieldLogo}})
	basicErr, ok := AsBasicCompanyError(err)
	if !ok || basicErr.Code != CodeBasicCompanyRequired || len(basicErr.Missing) != 1 {
		t.Fatalf("unexpected %#v %v", basicErr, ok)
	}
}
