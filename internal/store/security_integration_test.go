package store

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"wesmk/internal/db"
	"wesmk/internal/model"
	"wesmk/internal/secure"
)

func TestSecurityDataAtRest(t *testing.T) {
	dsn := os.Getenv("WESMK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WESMK_TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cipher, err := secure.NewCipher("kunci-integrasi-mfa-yang-panjang-123456")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	service := NewPostgres(pool, WithMFACipher(cipher))
	suffix := time.Now().UnixNano()
	userID := fmt.Sprintf("itest-security-%d", suffix)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	if err := service.CreateAccount(ctx, &model.User{
		ID: userID, Username: fmt.Sprintf("itest-sec-%d", suffix), FullName: "Admin Uji",
		Role: "admin_tu", PasswordHash: "hash", IsActive: true,
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if ok, err := service.SetTOTP(ctx, userID, secret, true); err != nil || !ok {
		t.Fatalf("set totp: ok=%v err=%v", ok, err)
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT totp_secret FROM users WHERE id = $1`, userID).Scan(&stored); err != nil {
		t.Fatalf("read stored: %v", err)
	}
	if !strings.HasPrefix(stored, secure.MFAPrefix) || strings.Contains(stored, secret) {
		t.Fatalf("secret is not encrypted at rest: %q", stored)
	}
	user, err := service.UserByID(ctx, userID)
	if err != nil || user == nil || user.TOTPSecret != secret {
		t.Fatalf("decrypted user = %+v err=%v", user, err)
	}

	// Simulate a legacy plaintext row and ensure the startup pass rewrites it.
	if _, err := pool.Exec(ctx, `UPDATE users SET totp_secret = $2 WHERE id = $1`, userID, secret); err != nil {
		t.Fatalf("legacy update: %v", err)
	}
	count, err := service.EncryptLegacyTOTPSecrets(ctx)
	if err != nil || count < 1 {
		t.Fatalf("re-encrypt = %d, %v", count, err)
	}

	hashes := []string{"hash-one", "hash-two"}
	if err := service.ReplaceRecoveryCodes(ctx, userID, hashes); err != nil {
		t.Fatalf("replace recovery: %v", err)
	}
	if count, err := service.CountUnusedRecoveryCodes(ctx, userID); err != nil || count != 2 {
		t.Fatalf("count codes = %d, %v", count, err)
	}
	if ok, err := service.ConsumeRecoveryCode(ctx, userID, "hash-one"); err != nil || !ok {
		t.Fatalf("consume = %v, %v", ok, err)
	}
	if ok, _ := service.ConsumeRecoveryCode(ctx, userID, "hash-one"); ok {
		t.Fatal("recovery code must be single-use")
	}
	if count, _ := service.CountUnusedRecoveryCodes(ctx, userID); count != 1 {
		t.Fatalf("unused codes = %d, want 1", count)
	}
	if err := service.DeleteRecoveryCodes(ctx, userID); err != nil {
		t.Fatalf("delete recovery codes: %v", err)
	}
	if count, _ := service.CountUnusedRecoveryCodes(ctx, userID); count != 0 {
		t.Fatalf("unused codes after delete = %d, want 0", count)
	}

	event := model.LoginEvent{
		UserID: userID, Identifier: "itest", Role: "admin_tu", Method: "totp",
		Success: true, IP: "10.0.0.1", UserAgent: "go-test",
	}
	if err := service.RecordLoginEvent(ctx, event); err != nil {
		t.Fatalf("record login: %v", err)
	}
	events, err := service.LoginEventsForUser(ctx, userID, 5)
	if err != nil || len(events) != 1 || events[0].Method != "totp" {
		t.Fatalf("login events = %+v, %v", events, err)
	}
}
