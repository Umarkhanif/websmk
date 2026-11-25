package state

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReversalOTPIsSingleUseAndNotPersisted(t *testing.T) {
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	doc := map[string]any{
		"ledger": []any{
			map[string]any{"id": float64(1001), "invoiceId": "inv-1", "amountDelta": float64(350000), "studentName": "Budi"},
			map[string]any{"id": float64(1002), "invoiceId": "inv-2", "amountDelta": float64(200000), "studentName": "Siti"},
		},
		"reversalRequests": []any{
			map[string]any{"id": "req-1", "ledgerId": float64(1001), "status": "MENUNGGU_OTP"},
		},
	}

	otp := GenerateReversalOTP(doc, "req-1", now)
	if len(otp) != 6 {
		t.Fatalf("otp = %q, want six digits", otp)
	}

	if err := ExecuteReversal(doc, 1001, "salah input nominal", otp, "usr-b", "Bendahara", "secret", now); err != nil {
		t.Fatalf("first reversal: %v", err)
	}
	if _, exists := doc["activeHeadmasterOtp"]; exists {
		t.Fatal("active OTP must be cleared after use")
	}
	if _, exists := doc["activeHeadmasterOtpExpiresAt"]; exists {
		t.Fatal("OTP expiry must be cleared after use")
	}

	reversal := asMap(asList(doc["ledger"])[0])
	if text(reversal["txType"]) != "REVERSAL_VOID" {
		t.Fatalf("newest ledger entry = %q, want REVERSAL_VOID", text(reversal["txType"]))
	}
	if strings.Contains(text(reversal["reversalApprovalId"]), otp) {
		t.Fatal("ledger must not store the OTP")
	}

	request := asMap(asList(doc["reversalRequests"])[0])
	if text(request["status"]) != "SELESAI_DIBUKUKAN" {
		t.Fatalf("request status = %q", text(request["status"]))
	}
	if _, exists := request["otpCode"]; exists {
		t.Fatal("request must not keep the consumed OTP")
	}

	if err := ExecuteReversal(doc, 1002, "percobaan kedua", otp, "usr-b", "Bendahara", "secret", now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("second reversal error = %v, want ErrInvalid", err)
	}
}
