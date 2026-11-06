package blstorage

import (
	"errors"
	"testing"
)

func TestFirebaseObjectName(t *testing.T) {
	u := "https://firebasestorage.googleapis.com/v0/b/kdauth-9893b.firebasestorage.app/o/companies%2Fc1%2Fproducts%2Fp1%2Fa.png?alt=media&token=abc"
	name, ok := firebaseObjectName(u, "kdauth-9893b.firebasestorage.app")
	if !ok || name != "companies/c1/products/p1/a.png" {
		t.Fatalf("got %q %v", name, ok)
	}
	if _, ok := firebaseObjectName(u, "other-bucket.appspot.com"); ok {
		t.Fatal("other bucket must be rejected")
	}
	for _, bad := range []string{
		"http://firebasestorage.googleapis.com/v0/b/x/o/companies%2Fc1%2Fa.png",
		"https://evil.example.com/v0/b/x/o/companies%2Fc1%2Fa.png",
		"https://firebasestorage.googleapis.com/other/path",
		"javascript:alert(1)",
	} {
		if _, ok := firebaseObjectName(bad, ""); ok {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestValidateCompanyFile(t *testing.T) {
	good := "https://firebasestorage.googleapis.com/v0/b/b1/o/companies%2Fc1%2Forders%2Fo1%2Ff.pdf?alt=media&token=t"
	if err := ValidateCompanyFile("c1", "companies/c1/orders/o1/f.pdf", good); err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	cases := map[string][3]string{
		"other company path": {"c2", "companies/c1/orders/o1/f.pdf", ""},
		"dot dot":            {"c1", "companies/c1/../c2/f.pdf", ""},
		"other company url":  {"c2", "", good},
		"path/url mismatch":  {"c1", "companies/c1/orders/o1/other.pdf", good},
		"not firebase":       {"c1", "", "https://example.com/companies/c1/f.pdf"},
	}
	for name, c := range cases {
		if err := ValidateCompanyFile(c[0], c[1], c[2]); !errors.Is(err, ErrInvalidStorageFile) {
			t.Errorf("%s: expected ErrInvalidStorageFile, got %v", name, err)
		}
	}
}
