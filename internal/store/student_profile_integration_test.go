package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

// TestStudentProfileDomainCutover runs against a real Postgres when
// WESMK_TEST_DATABASE_URL is set (CI uses a disposable container).
func TestStudentProfileDomainCutover(t *testing.T) {
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
	// Boot runs the migrator every time; it must be a no-op the second time.
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate twice: %v", err)
	}

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = 'itest-user-profile'`)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash)
		VALUES ('itest-user-profile', 'itest-profile-identifier', 'Nama Akun', 'student', 'x')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Simulate a row created before migration 007: the boot backfill must fill
	// the new profile columns without touching finance data.
	if _, err := pool.Exec(ctx, `
		INSERT INTO students (user_id, nisn) VALUES ('itest-user-profile', '0099999999')
	`); err != nil {
		t.Fatalf("insert legacy student: %v", err)
	}

	doc := map[string]any{
		"students": []any{map[string]any{
			"id":              "itest-std-profile",
			"userId":          "itest-user-profile",
			"nisn":            "0099999999",
			"nik":             "3201284509050001",
			"fullName":        "Nama dari Blob",
			"classroomName":   "XII TKJ 1",
			"majorCode":       "TKJ",
			"parentName":      "Bapak Uji",
			"parentPhone":     "081200000000",
			"isKetuaKelas":    true,
			"scholarshipType": "NONE",
			"discountPercent": float64(0),
			"attitudeScore":   float64(92),
			"bmwChoice":       "BEKERJA",
		}},
		"invoices": []any{},
		"ledger":   []any{},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	service := NewPostgres(pool)
	if err := service.PutState(ctx, raw); err != nil {
		t.Fatalf("put state: %v", err)
	}
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	profile, err := service.StudentProfile(ctx, "itest-std-profile")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if profile.FullName != "Nama dari Blob" || profile.NISN != "0099999999" ||
		profile.ClassroomName != "XII TKJ 1" || profile.ParentName != "Bapak Uji" ||
		!profile.IsKetuaKelas || profile.AttitudeScore != 92 || profile.BMWChoice != "BEKERJA" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	// Shadow sync after a blob mutation must make the table current.
	doc["students"].([]any)[0].(map[string]any)["fullName"] = "Nama Setelah Ubah"
	doc["students"].([]any)[0].(map[string]any)["attitudeScore"] = float64(95)
	if err := service.SyncDomainFromState(ctx, doc); err != nil {
		t.Fatalf("sync: %v", err)
	}
	profile, err = service.StudentProfile(ctx, "itest-std-profile")
	if err != nil {
		t.Fatalf("profile after sync: %v", err)
	}
	if profile.FullName != "Nama Setelah Ubah" || profile.AttitudeScore != 95 {
		t.Fatalf("sync did not update profile: %+v", profile)
	}
}
