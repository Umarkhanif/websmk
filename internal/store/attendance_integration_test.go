package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

func TestAttendanceDomainCutover(t *testing.T) {
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
	userID := fmt.Sprintf("itest-att-user-%d", suffix)
	studentAppID := fmt.Sprintf("itest-att-std-%d", suffix)
	nisn := fmt.Sprintf("%010d", suffix%10000000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash)
		VALUES ($1, $1, 'Siswa Presensi', 'student', 'x')
	`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	doc := map[string]any{
		"students": []any{map[string]any{
			"id": studentAppID, "userId": userID, "nisn": nisn,
			"fullName": "Siswa Presensi",
		}},
		"attendances": []any{
			map[string]any{
				"id": "itest-att-1-" + fmt.Sprint(suffix), "studentId": studentAppID,
				"studentName": "Siswa Presensi", "scheduleId": "sch-1",
				"status": "HADIR_GERBANG", "verifiedSource": "RFID Gerbang",
				"updatedAt": "2026-11-23T06:48:15Z",
			},
			map[string]any{
				"id": "itest-att-2-" + fmt.Sprint(suffix), "studentId": studentAppID,
				"studentName": "Siswa Presensi", "scheduleId": "sch-2",
				"status": "IZIN", "verifiedSource": "Form Orang Tua",
				"updatedAt": "2026-11-22T07:00:00Z",
			},
		},
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

	records, err := service.AttendanceForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("attendance: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("attendance = %d, want 2", len(records))
	}
	if records[0].Status != "HADIR_GERBANG" || records[0].VerifiedSource != "RFID Gerbang" ||
		records[0].NISN != nisn || records[0].StudentAppID != studentAppID {
		t.Fatalf("unexpected newest record: %+v", records[0])
	}

	all, err := service.AllAttendances(ctx)
	if err != nil {
		t.Fatalf("all attendance: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("all attendance = %d, want at least 2", len(all))
	}

	// Shadow sync updates status and appends a third day without duplicating.
	doc["attendances"].([]any)[0].(map[string]any)["status"] = "ALPA"
	doc["attendances"] = append(doc["attendances"].([]any), map[string]any{
		"id": "itest-att-3-" + fmt.Sprint(suffix), "studentId": studentAppID,
		"studentName": "Siswa Presensi", "scheduleId": "sch-3",
		"status": "SAKIT", "verifiedSource": "Surat Dokter",
		"updatedAt": "2026-11-24T06:30:00Z",
	})
	if err := service.SyncDomainFromState(ctx, doc); err != nil {
		t.Fatalf("sync: %v", err)
	}
	records, err = service.AttendanceForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("attendance after sync: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("attendance after sync = %d, want 3", len(records))
	}
	if records[0].Status != "SAKIT" {
		t.Fatalf("newest status = %q, want SAKIT", records[0].Status)
	}

	// Boot backfill is idempotent.
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("re-backfill: %v", err)
	}
	records, err = service.AttendanceForStudent(ctx, studentAppID)
	if err != nil || len(records) != 3 {
		t.Fatalf("attendance after re-backfill = %d, err %v", len(records), err)
	}
}
