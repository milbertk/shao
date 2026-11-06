package blpublicshowcase

import (
	"errors"
	"strings"
	"testing"
)

func TestValidPublicPhone(t *testing.T) {
	ok := []string{"", "+506 8888-8888", "(506) 2222.3333", "88888888", "+1 (555) 123-4567"}
	bad := []string{"123", "abc12345678", "+506 8888 8888 8888 8888", "88888888<script>", "506+88888888"}
	for _, v := range ok {
		if !ValidPublicPhone(v) {
			t.Errorf("phone %q should be valid", v)
		}
	}
	for _, v := range bad {
		if ValidPublicPhone(v) {
			t.Errorf("phone %q should be invalid", v)
		}
	}
}

func TestValidatePublicWebOrderRequest(t *testing.T) {
	good := PublicWebOrderRequest{CustomerName: "Ana", Email: "ana@example.com", Phone: "8888-8888",
		Items: []PublicWebOrderItem{{ProductID: "p1", Quantity: 2}}}
	if err := ValidatePublicWebOrderRequest(good); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	cases := map[string]PublicWebOrderRequest{
		"bad email":   {CustomerName: "Ana", Email: "ana@", Items: good.Items},
		"header inj.": {CustomerName: "Ana", Email: "ana@example.com\r\nBcc: x@y.z", Items: good.Items},
		"bad phone":   {CustomerName: "Ana", Phone: "12", Items: good.Items},
		"long name":   {CustomerName: strings.Repeat("a", 121), Items: good.Items},
		"huge qty":    {CustomerName: "Ana", Items: []PublicWebOrderItem{{ProductID: "p1", Quantity: 5000}}},
		"long notes":  {CustomerName: "Ana", Notes: strings.Repeat("n", 2001), Items: good.Items},
		"too many":    {CustomerName: "Ana", Items: make([]PublicWebOrderItem, 51)},
	}
	for name, req := range cases {
		err := ValidatePublicWebOrderRequest(req)
		if err == nil || !errors.Is(err, ErrPublicFormInvalid) {
			t.Errorf("%s: expected ErrPublicFormInvalid, got %v", name, err)
		}
	}
}

func TestValidatePublicAppointmentRequest(t *testing.T) {
	if err := ValidatePublicAppointmentRequest(PublicAppointmentRequest{CustomerName: "Luis", Email: "luis@example.com"}); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if err := ValidatePublicAppointmentRequest(PublicAppointmentRequest{CustomerName: "Luis", Email: "not-an-email"}); !errors.Is(err, ErrPublicFormInvalid) {
		t.Fatalf("bad email accepted: %v", err)
	}
}
