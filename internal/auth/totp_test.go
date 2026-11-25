package auth

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 test vector (SHA-1, 6-digit truncation) at T=59.
func TestTOTPMatchesRFCVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := TOTPCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if code != "287082" {
		t.Fatalf("code = %q, want 287082", code)
	}
	if !ValidateTOTP(secret, code, time.Unix(59, 0)) {
		t.Fatal("current code must validate")
	}
	if ValidateTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("wrong code must not validate")
	}
}

func TestTOTPAllowsOneStepDrift(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	now := time.Unix(1_700_000_000, 0)
	previous, err := TOTPCode(secret, now.Add(-30*time.Second))
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if !ValidateTOTP(secret, previous, now) {
		t.Fatal("previous step should be accepted")
	}
}

func TestTOTPURIContainsIssuerAndSecret(t *testing.T) {
	uri := TOTPURI("ABC234", "199201052019032011", "WESMK")
	for _, part := range []string{"otpauth://totp/", "secret=ABC234", "issuer=WESMK"} {
		if !strings.Contains(uri, part) {
			t.Fatalf("uri %q missing %q", uri, part)
		}
	}
}
