package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
	"wesmk/internal/domain/pkl"
)

func TestPklBkkDomainCutover(t *testing.T) {
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
	userID := fmt.Sprintf("itest-pkl-user-%d", suffix)
	studentAppID := fmt.Sprintf("itest-pkl-std-%d", suffix)
	placementAppID := fmt.Sprintf("itest-pkl-place-%d", suffix)
	logAppID := fmt.Sprintf("itest-pkl-log-%d", suffix)
	journalAppID := fmt.Sprintf("itest-pkl-journal-%d", suffix)
	jobAppID := fmt.Sprintf("itest-bkk-job-%d", suffix)
	appAppID := fmt.Sprintf("itest-bkk-app-%d", suffix)
	nisn := fmt.Sprintf("%010d", suffix%10000000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM students WHERE nisn = $1`, nisn)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash)
		VALUES ($1, $1, 'Siswa PKL', 'student', 'x')
	`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	doc := map[string]any{
		"students": []any{map[string]any{
			"id": studentAppID, "userId": userID, "nisn": nisn, "fullName": "Siswa PKL",
			"classroomName": "XII TKJ 1", "majorCode": "TKJ", "parentName": "Orang Tua", "parentPhone": "0812",
		}},
		"pklPlacements": []any{map[string]any{
			"id": placementAppID, "studentId": studentAppID,
			"companyName": "PT Jaringan Nusantara", "industrySector": "Telekomunikasi",
			"address": "Jl. Fiber No. 1", "gpsLat": float64(-6.2), "gpsLng": float64(106.8),
			"radiusMeters": float64(150), "supervisorName": "Pak Budi",
			"supervisorPhone": "0813", "supervisorPin": "hashed-pin",
		}},
		"pklAttendanceLogs": []any{map[string]any{
			"id": logAppID, "placementId": placementAppID, "studentId": studentAppID,
			"studentName": "Siswa PKL", "checkinTime": "2026-11-20T08:00:00Z",
			"gpsLat": float64(-6.2), "gpsLng": float64(106.8), "distanceMeters": float64(12),
			"isWithinRadius": true, "selfieUrl": "blob://selfie",
		}},
		"pklJournals": []any{map[string]any{
			"id": journalAppID, "placementId": placementAppID, "studentId": studentAppID,
			"studentName": "Siswa PKL", "companyName": "PT Jaringan Nusantara",
			"date": "2026-11-20", "activityTitle": "Konfigurasi OLT",
			"competencyLearned": "Splicing fiber", "workPhotoUrl": "blob://photo",
			"supervisorApproved": true, "teacherApproved": false,
		}},
		"bkkJobs": []any{map[string]any{
			"id": jobAppID, "companyName": "PT Jaringan Nusantara", "jobTitle": "Teknisi Fiber",
			"targetedMajors": []any{"TKJ"}, "salaryRange": "Rp4.000.000",
			"deadlineDate": "2026-12-31", "requirements": []any{"Jujur", "Disiplin"},
			"applicantCount": float64(3),
		}},
		"bkkApplications": []any{map[string]any{
			"id": appAppID, "jobId": jobAppID, "studentId": studentAppID,
			"studentName": "Siswa PKL", "appliedDate": "2026-11-21", "status": "SELEKSI",
		}},
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

	placements, err := service.PklPlacementsForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("placements: %v", err)
	}
	if len(placements) != 1 || placements[0].CompanyName != "PT Jaringan Nusantara" {
		t.Fatalf("unexpected placements: %+v", placements)
	}
	if placements[0].SupervisorPin != "hashed-pin" {
		t.Fatalf("store should expose stored pin hash, got %q", placements[0].SupervisorPin)
	}
	svc := pkl.NewService(service)
	redacted, err := svc.PlacementsForStudent(ctx, pkl.Viewer{Role: "student", StudentID: studentAppID}, studentAppID)
	if err != nil {
		t.Fatalf("service placements: %v", err)
	}
	if redacted[0].SupervisorPin != "" {
		t.Fatalf("student must not receive supervisor pin, got %q", redacted[0].SupervisorPin)
	}

	logs, err := service.PklLogsForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if len(logs) != 1 || logs[0].DistanceMeters != 12 || !logs[0].IsWithinRadius || logs[0].StudentName != "Siswa PKL" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	if logs[0].CheckinTime != "2026-11-20T08:00:00Z" {
		t.Fatalf("unexpected checkin time: %q", logs[0].CheckinTime)
	}

	journals, err := service.PklJournalsForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("journals: %v", err)
	}
	if len(journals) != 1 || journals[0].ActivityTitle != "Konfigurasi OLT" || !journals[0].SupervisorApproved || journals[0].TeacherApproved {
		t.Fatalf("unexpected journals: %+v", journals)
	}

	jobs, err := service.BkkJobs(ctx)
	if err != nil {
		t.Fatalf("jobs: %v", err)
	}
	var foundJob bool
	for _, job := range jobs {
		if job.AppID == jobAppID {
			foundJob = true
			if job.JobTitle != "Teknisi Fiber" || len(job.Requirements) != 2 || job.ApplicantCount != 3 {
				t.Fatalf("unexpected job: %+v", job)
			}
		}
	}
	if !foundJob {
		t.Fatalf("job %q not found in %d rows", jobAppID, len(jobs))
	}

	apps, err := service.BkkApplicationsForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("applications: %v", err)
	}
	if len(apps) != 1 || apps[0].Status != "SELEKSI" || apps[0].JobTitle != "Teknisi Fiber" || apps[0].AppliedDate != "2026-11-21" {
		t.Fatalf("unexpected applications: %+v", apps)
	}

	// A second sync must stay idempotent and keep one row per app id.
	if err := service.SyncDomainFromState(ctx, doc); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if rows, err := service.PklJournalsForStudent(ctx, studentAppID); err != nil || len(rows) != 1 {
		t.Fatalf("journals after resync: %d rows, err %v", len(rows), err)
	}
}
