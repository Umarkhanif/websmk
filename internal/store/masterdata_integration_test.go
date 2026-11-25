package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

func TestRolloverStudentsPromotesAndGraduatesAtomically(t *testing.T) {
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

	service := NewPostgres(pool)
	suffix := time.Now().UnixNano() % 1_000_000
	previous, hadPrevious := service.ActiveAcademicYear(ctx)
	year, err := service.CreateAcademicYear(ctx,
		fmt.Sprintf("%04d/%04d", 3000+suffix%900, 3001+suffix%900), "Ganjil",
		time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create target year: %v", err)
	}

	classX, err := service.CreateClassroom(ctx, fmt.Sprintf("X Rollover %d", suffix), 10, "TKJ", "")
	if err != nil {
		t.Fatalf("create X: %v", err)
	}
	classXI, err := service.CreateClassroom(ctx, fmt.Sprintf("XI Rollover %d", suffix), 11, "TKJ", "")
	if err != nil {
		t.Fatalf("create XI: %v", err)
	}
	classXII, err := service.CreateClassroom(ctx, fmt.Sprintf("XII Rollover %d", suffix), 12, "TKJ", "")
	if err != nil {
		t.Fatalf("create XII: %v", err)
	}

	students := []struct {
		userID, appID string
		nisn          string
		classID       string
		className     string
	}{
		{fmt.Sprintf("usr-roll-x-%d", suffix), fmt.Sprintf("roll-x-%d", suffix), fmt.Sprintf("%010d", suffix), classX.ID, classX.Name},
		{fmt.Sprintf("usr-roll-xi-%d", suffix), fmt.Sprintf("roll-xi-%d", suffix), fmt.Sprintf("%010d", suffix+1), classXI.ID, classXI.Name},
		{fmt.Sprintf("usr-roll-xii-%d", suffix), fmt.Sprintf("roll-xii-%d", suffix), fmt.Sprintf("%010d", suffix+2), classXII.ID, classXII.Name},
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, student := range students {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, student.userID)
		}
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM classrooms WHERE id IN ($1::uuid, $2::uuid, $3::uuid)`, classX.ID, classXI.ID, classXII.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM academic_years WHERE id = $1::uuid`, year.ID)
		if hadPrevious {
			_, _ = pool.Exec(cleanupCtx, `UPDATE academic_years SET is_active = FALSE WHERE is_active`)
			_, _ = pool.Exec(cleanupCtx, `UPDATE academic_years SET is_active = TRUE WHERE id = $1::uuid`, previous.ID)
		}
	})

	for _, student := range students {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (id, identifier, full_name, role, password_hash, is_active)
			VALUES ($1, $2, 'Siswa Rollover', 'student', 'x', TRUE)
		`, student.userID, student.nisn); err != nil {
			t.Fatalf("insert student user: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO students (app_id, user_id, nisn, full_name, classroom_id, classroom_name, major_code)
			VALUES ($1, $2, $3, 'Siswa Rollover', $4::uuid, $5, 'TKJ')
		`, student.appID, student.userID, student.nisn, student.classID, student.className); err != nil {
			t.Fatalf("insert student: %v", err)
		}
	}

	if _, err := service.RolloverStudents(ctx, year.ID, []RolloverMove{{FromClassID: classX.ID, ToClassID: classX.ID}}, nil); err == nil {
		t.Fatal("rollover into the same class should fail")
	}
	if active, ok := service.ActiveAcademicYear(ctx); hadPrevious && (!ok || active.ID != previous.ID) {
		t.Fatal("failed rollover must not activate the target year")
	}

	result, err := service.RolloverStudents(ctx, year.ID, []RolloverMove{
		{FromClassID: classX.ID, ToClassID: classXI.ID},
		{FromClassID: classXI.ID, ToClassID: classXII.ID},
	}, []string{classXII.ID})
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if result.MovedStudents != 2 || result.GraduatedStudents != 1 {
		t.Fatalf("rollover result = %+v, want 2 moved and 1 graduated", result)
	}

	var promotedName string
	if err := pool.QueryRow(ctx, `SELECT classroom_name FROM students WHERE app_id = $1`, students[0].appID).Scan(&promotedName); err != nil {
		t.Fatalf("read promoted student: %v", err)
	}
	if promotedName != classXI.Name {
		t.Fatalf("promoted classroom = %q, want %q", promotedName, classXI.Name)
	}
	var finalName string
	if err := pool.QueryRow(ctx, `SELECT classroom_name FROM students WHERE app_id = $1`, students[1].appID).Scan(&finalName); err != nil {
		t.Fatalf("read final-year student: %v", err)
	}
	if finalName != classXII.Name {
		t.Fatalf("final-year classroom = %q, want %q", finalName, classXII.Name)
	}
	var alumni bool
	if err := pool.QueryRow(ctx, `SELECT is_alumni FROM students WHERE app_id = $1`, students[2].appID).Scan(&alumni); err != nil {
		t.Fatalf("read graduate: %v", err)
	}
	if !alumni {
		t.Fatal("grade 12 student should be an alumni")
	}
	if active, ok := service.ActiveAcademicYear(ctx); !ok || active.ID != year.ID {
		t.Fatalf("active year = %+v, want target %s", active, year.ID)
	}
}
