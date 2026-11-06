package blpageconfiguration

import "testing"

func TestNormalizeSlug(t *testing.T) {
	cases := map[string]string{
		"Maderas Finas":         "maderas-finas",
		"  Café  Ñandú  ":       "cafe-nandu",
		"Taller_de.Motos 2":     "taller-de-motos-2",
		"--Hola!!  mundo--":     "hola-mundo",
		"ÁÉÍÓÚ üñ":              "aeiou-un",
		"La Tienda de Doña Ana": "la-tienda-de-dona-ana",
	}
	for in, want := range cases {
		if got := NormalizeSlug(in); got != want {
			t.Errorf("NormalizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	good := []string{"maderas-finas", "abc", "tienda-2"}
	for _, v := range good {
		if err := ValidateSlug(v); err != nil {
			t.Errorf("ValidateSlug(%q) unexpected error %v", v, err)
		}
	}
	bad := []string{"ab", "api", "showcase", "Maderas", "a--b", "-abc",
		"fb47a31b-f412-442d-bad0-ce077eb0dffe"}
	for _, v := range bad {
		if err := ValidateSlug(v); err == nil {
			t.Errorf("ValidateSlug(%q) expected an error", v)
		}
	}
}

func TestLooksLikeCompanyID(t *testing.T) {
	if !LooksLikeCompanyID("fb47a31b-f412-442d-bad0-ce077eb0dffe") {
		t.Fatal("uuid not detected")
	}
	if LooksLikeCompanyID("maderas-finas") {
		t.Fatal("slug detected as id")
	}
}
