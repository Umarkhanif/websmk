package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
	"wesmk/internal/model"
)

func TestAccountLifecycle(t *testing.T) {
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
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	suffix := time.Now().UnixNano()
	userID := fmt.Sprintf("itest-account-%d", suffix)
	identifier := fmt.Sprintf("itest-%d", suffix)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	service := NewPostgres(pool)
	account := &model.User{
		ID: userID, Username: identifier, FullName: "Guru Uji", Role: "teacher",
		PhoneWA: "081200000000", PasswordHash: "hash-a", IsActive: true,
	}
	if err := service.CreateAccount(ctx, account); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := service.CreateAccount(ctx, account); !errors.Is(err, ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got %v", err)
	}

	users, err := service.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *model.User
	for _, user := range users {
		if user.ID == userID {
			found = user
		}
	}
	if found == nil || !found.IsActive || found.Role != "teacher" {
		t.Fatalf("unexpected account: %+v", found)
	}

	if ok, err := service.SetUserActive(ctx, userID, false); err != nil || !ok {
		t.Fatalf("deactivate: ok=%v err=%v", ok, err)
	}
	if found, _ = service.UserByID(ctx, userID); found == nil || found.IsActive {
		t.Fatalf("account should be inactive: %+v", found)
	}

	if ok, err := service.SetUserPasswordHash(ctx, userID, "hash-b"); err != nil || !ok {
		t.Fatalf("password: ok=%v err=%v", ok, err)
	}
	if found, _ = service.UserByID(ctx, userID); found.PasswordHash != "hash-b" {
		t.Fatalf("password not updated: %+v", found)
	}

	if err := service.RecordExecutiveSignature(ctx, userID, "laporan-eksekutif", "hash-doc", "hmac-doc"); err != nil {
		t.Fatalf("record signature: %v", err)
	}
	signature, err := service.LatestExecutiveSignature(ctx)
	if err != nil || signature == nil || signature.DocumentHash != "hash-doc" || signature.SignatureHMAC != "hmac-doc" {
		t.Fatalf("unexpected signature: %+v err=%v", signature, err)
	}
}
