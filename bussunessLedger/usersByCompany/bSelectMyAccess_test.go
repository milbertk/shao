package usersbycompany

import "testing"

func TestBSelectMyAccessFullRoles(t *testing.T) {
	for _, role := range []string{"OWNER", "manager", " Owner "} {
		result, err := BSelectMyAccess("company-1", "user-1", role)
		if err != nil {
			t.Fatalf("role %q: unexpected error %v", role, err)
		}
		if !result.FullAccess || len(result.Permissions) != 0 {
			t.Fatalf("role %q: expected full access, got %+v", role, result)
		}
	}
}

func TestBSelectMyAccessRequiresIDs(t *testing.T) {
	if _, err := BSelectMyAccess("", "user-1", "OWNER"); err == nil {
		t.Fatal("expected an error without companyId")
	}
	if _, err := BSelectMyAccess("company-1", " ", "OWNER"); err == nil {
		t.Fatal("expected an error without user")
	}
}
