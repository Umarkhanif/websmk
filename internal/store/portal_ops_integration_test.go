package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

func TestPortalOperationsUseNormalizedData(t *testing.T) {
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
	teacherID := fmt.Sprintf("usr-teacher-uji-%d", suffix)
	studentUserID := fmt.Sprintf("usr-student-uji-%d", suffix)
	studentAppID := fmt.Sprintf("std-uji-%d", suffix)
	nisn := fmt.Sprintf("%010d", suffix)
	classID := ""
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM gate_logs WHERE nisn = $1`, nisn)
		_, _ = pool.Exec(context.Background(), `DELETE FROM substitute_assignments WHERE teacher_id = $1`, teacherID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM class_journals WHERE teacher_id = $1`, teacherID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM schedules WHERE classroom_id = NULLIF($1, '')::uuid`, classID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1, $2)`, teacherID, studentUserID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM classrooms WHERE id = NULLIF($1, '')::uuid`, classID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash, is_active)
		VALUES ($1, $2, 'Guru Uji', 'teacher', 'x', TRUE)
	`, teacherID, "NIP-"+teacherID); err != nil {
		t.Fatalf("insert teacher: %v", err)
	}
	class, err := service.CreateClassroom(ctx, fmt.Sprintf("X Uji %d", suffix), 10, "TKJ", teacherID)
	if err != nil {
		t.Fatalf("create classroom: %v", err)
	}
	classID = class.ID
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash, is_active)
		VALUES ($1, $2, 'Siswa Uji', 'student', 'x', TRUE)
	`, studentUserID, nisn); err != nil {
		t.Fatalf("insert student user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO students (app_id, user_id, nisn, full_name, classroom_id, classroom_name, major_code)
		VALUES ($1, $2, $3, 'Siswa Uji', $4, $5, 'TKJ')
	`, studentAppID, studentUserID, nisn, class.ID, class.Name); err != nil {
		t.Fatalf("insert student: %v", err)
	}

	date := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	roster, err := service.ClassroomRoster(ctx, class.ID, date)
	if err != nil || len(roster) != 1 || roster[0].StudentAppID != studentAppID {
		t.Fatalf("roster = %+v, %v", roster, err)
	}
	allowed, err := service.TeacherCanManageClassroom(ctx, teacherID, class.ID, date)
	if err != nil || !allowed {
		t.Fatalf("teacher access = %v, %v", allowed, err)
	}

	if _, err := service.UpsertStudentAttendance(ctx, studentAppID, date, "ALPA_BOLOS", "uji"); err != nil {
		t.Fatalf("upsert attendance: %v", err)
	}
	roster, err = service.ClassroomRoster(ctx, class.ID, date)
	if err != nil || roster[0].Status != "ALPA_BOLOS" {
		t.Fatalf("roster status = %+v, %v", roster, err)
	}

	if err := service.CreateSchedule(ctx, class.ID, "Matematika", teacherID, int(date.Weekday()), "08:00", "09:30", "Lab Uji"); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	schedules, err := service.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	scheduleID := ""
	for _, schedule := range schedules {
		if schedule.ClassroomID == class.ID {
			scheduleID = schedule.ID
		}
	}
	if scheduleID == "" {
		t.Fatal("schedule not found")
	}
	if err := service.SaveClassJournal(ctx, scheduleID, teacherID, date, "Topik Uji", "Catatan Uji", "Modul Uji"); err != nil {
		t.Fatalf("save journal: %v", err)
	}
	journal, ok := service.ClassJournalForScheduleDate(ctx, scheduleID, date)
	if !ok || journal.Topic != "Topik Uji" || journal.SmartNotes != "Modul Uji" {
		t.Fatalf("journal = %+v, %v", journal, ok)
	}

	leave, err := service.CreateParentLeaveRequest(ctx, studentAppID, "SAKIT", date, "Demam")
	if err != nil || leave.LeaveDate != "2026-11-24" || leave.Status != "TERCATAT" {
		t.Fatalf("leave = %+v, %v", leave, err)
	}
	leaves, err := service.ParentLeavesForStudent(ctx, studentAppID)
	if err != nil || len(leaves) != 1 || leaves[0].Note != "Demam" {
		t.Fatalf("leaves = %+v, %v", leaves, err)
	}

	log, err := service.CreateGateLog(ctx, nisn, "TERLAMBAT_MASUK", "Uji", "PASS-UJI", teacherID)
	if err != nil || log.StudentName != "Siswa Uji" {
		t.Fatalf("gate log = %+v, %v", log, err)
	}
	if _, err := service.CreateSubstituteAssignment(ctx, scheduleID, teacherID, "Titipan", teacherID, date); err != nil {
		t.Fatalf("substitute: %v", err)
	}
	assignments, err := service.SubstituteAssignmentsForDate(ctx, date)
	foundAssignment := false
	for _, assignment := range assignments {
		if assignment.ScheduleID == scheduleID && assignment.ClassName == class.Name {
			foundAssignment = true
		}
	}
	if err != nil || !foundAssignment {
		t.Fatalf("assignments = %+v, %v", assignments, err)
	}

	rows, total, err := service.StudentDirectory(ctx, nisn, 10, 0)
	if err != nil || total != 1 || rows[0].FullName != "Siswa Uji" {
		t.Fatalf("directory = %+v, %d, %v", rows, total, err)
	}

	updated, err := service.UpdateClassroom(ctx, class.ID, class.Name+" Baru", 11, "TKJ", teacherID)
	if err != nil || updated.Name != class.Name+" Baru" || updated.GradeLevel != 11 {
		t.Fatalf("updated classroom = %+v, %v", updated, err)
	}
	profile, err := service.StudentProfile(ctx, studentAppID)
	if err != nil || profile.ClassroomName != class.Name+" Baru" || profile.ClassroomID != class.ID {
		t.Fatalf("student classroom sync = %+v, %v", profile, err)
	}
	if err := service.UpdateSchedule(ctx, scheduleID, class.ID, "Matematika", teacherID, int(date.Weekday()), "10:00", "11:30", "Lab Baru"); err != nil {
		t.Fatalf("update schedule: %v", err)
	}
	if err := service.DeleteClassroom(ctx, class.ID); !errors.Is(err, ErrMasterDataInUse) {
		t.Fatalf("delete occupied classroom = %v, want ErrMasterDataInUse", err)
	}
	if err := service.DeleteSchedule(ctx, scheduleID); err != nil {
		t.Fatalf("delete schedule: %v", err)
	}
	empty, err := service.CreateClassroom(ctx, fmt.Sprintf("XI Kosong %d", suffix), 11, "TKJ", "")
	if err != nil {
		t.Fatalf("create empty classroom: %v", err)
	}
	if err := service.DeleteClassroom(ctx, empty.ID); err != nil {
		t.Fatalf("delete empty classroom: %v", err)
	}
}
