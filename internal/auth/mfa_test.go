package auth

import (
	"context"
	"testing"
	"time"
)

func TestMFAChallengeRequiresValidTOTPAndIsSingleUse(t *testing.T) {
	service, repo, _ := newTestService(t, Options{})
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	user := repo.byIdentifier["0058471290"]
	user.TOTPSecret = secret
	user.TOTPEnabled = true

	ctx := context.Background()
	session, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.7")
	if session != nil || !IsErrorCode(err, CodeMFARequired) {
		t.Fatalf("login = %v, %v; want mfa challenge", session, err)
	}
	authErr, ok := err.(*Error)
	if !ok || authErr.Challenge == "" {
		t.Fatalf("challenge missing: %#v", err)
	}

	if _, err := service.VerifyMFA(ctx, authErr.Challenge, "000000"); err == nil {
		t.Fatal("wrong code must fail")
	}
	code, err := TOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	session, err = service.VerifyMFA(ctx, authErr.Challenge, code)
	if err != nil || session == nil || session.User.ID != user.ID {
		t.Fatalf("verify mfa = %v, %v", session, err)
	}
	if _, err := service.VerifyMFA(ctx, authErr.Challenge, code); err == nil {
		t.Fatal("challenge must be single use")
	}
}

func TestInactiveAccountCannotUseMFA(t *testing.T) {
	service, repo, _ := newTestService(t, Options{})
	secret, _ := GenerateTOTPSecret()
	user := repo.byIdentifier["0058471290"]
	user.TOTPSecret = secret
	user.TOTPEnabled = true

	ctx := context.Background()
	_, err := service.Login(ctx, "0058471290", testDemoPassword, "10.0.0.8")
	authErr, ok := err.(*Error)
	if !ok || authErr.Challenge == "" {
		t.Fatalf("expected challenge, got %v", err)
	}
	user.IsActive = false
	code, _ := TOTPCode(secret, time.Now())
	if _, err := service.VerifyMFA(ctx, authErr.Challenge, code); err == nil {
		t.Fatal("inactive account must not complete MFA")
	}
}
