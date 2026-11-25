package auth

import (
	"context"
	"testing"
)

func TestRecoveryCodeAuthenticatesOnce(t *testing.T) {
	service, repo, _ := newTestService(t, Options{})
	secret, _ := GenerateTOTPSecret()
	user := repo.byIdentifier["196805121994121001"]
	user.TOTPSecret = secret
	user.TOTPEnabled = true

	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("generate recovery codes: %v", err)
	}
	repo.recovery[user.ID+":"+RecoveryCodeHash(user.ID, codes[0])] = true

	ctx := context.Background()
	_, err = service.Login(ctx, user.Username, testDemoPassword, "10.0.0.9")
	challenge := err.(*Error).Challenge
	session, err := service.VerifyMFA(ctx, challenge, codes[0])
	if err != nil || session == nil {
		t.Fatalf("recovery login = %v, %v", session, err)
	}
	if session.Method != "recovery_code" {
		t.Fatalf("method = %q, want recovery_code", session.Method)
	}

	_, err = service.Login(ctx, user.Username, testDemoPassword, "10.0.0.9")
	challenge = err.(*Error).Challenge
	if _, err := service.VerifyMFA(ctx, challenge, codes[0]); err == nil {
		t.Fatal("a used recovery code must not work again")
	}
}

func TestPrivilegedRolesRequireMFA(t *testing.T) {
	_, repo, _ := newTestService(t, Options{})
	headmaster := repo.byIdentifier["196805121994121001"]
	if !RequiresMFA(headmaster) {
		t.Fatal("a real headmaster must require MFA")
	}
	headmaster.Role = "teacher"
	if RequiresMFA(headmaster) {
		t.Fatal("teacher must not be forced into MFA")
	}
}
