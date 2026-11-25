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

func TestCreateStudentWithAccounts(t *testing.T) {
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
	suffix := time.Now().UnixNano() % 10_000_000_000
	nisn := fmt.Sprintf("%010d", suffix)
	phone := fmt.Sprintf("0812%08d", suffix%100_000_000)
	input := StudentOnboarding{
		StudentID: fmt.Sprintf("itest-std-%d", suffix), StudentUserID: fmt.Sprintf("itest-usr-std-%d", suffix),
		ParentUserID: fmt.Sprintf("itest-usr-prt-%d", suffix), FullName: "Siswa Onboarding Uji",
		NISN: nisn, ClassroomName: "X TKJ 1", MajorCode: "TKJ", ParentName: "Wali Onboarding Uji",
		ParentPhone: phone, StudentPasswordHash: "student-hash", ParentPasswordHash: "parent-hash",
	}
	t.Cleanup(func() {
		_ = service.DeleteStudentOnboarding(context.Background(), input.StudentUserID, input.ParentUserID)
	})
	if err := service.CreateStudentWithAccounts(ctx, input); err != nil {
		t.Fatalf("create onboarding: %v", err)
	}

	student, err := service.UserByIdentifier(ctx, nisn)
	if err != nil || student == nil || student.Role != "student" || student.StudentProfileID != input.StudentID {
		t.Fatalf("student user = %+v, %v", student, err)
	}
	parent, err := service.ParentByPhone(ctx, phone)
	if err != nil || parent == nil || parent.Role != "parent" || parent.StudentProfileID != input.StudentID {
		t.Fatalf("parent user = %+v, %v", parent, err)
	}
	profile, err := service.StudentProfile(ctx, input.StudentID)
	if err != nil || profile.FullName != input.FullName || profile.ParentPhone != phone {
		t.Fatalf("student profile = %+v, %v", profile, err)
	}

	duplicateNISN := input
	duplicateNISN.StudentID = input.StudentID + "-dup"
	duplicateNISN.StudentUserID = input.StudentUserID + "-dup"
	duplicateNISN.ParentUserID = input.ParentUserID + "-dup"
	duplicateNISN.ParentPhone = fmt.Sprintf("0822%08d", suffix%100_000_000)
	if err := service.CreateStudentWithAccounts(ctx, duplicateNISN); !errors.Is(err, ErrStudentNISNExists) {
		t.Fatalf("duplicate NISN = %v, want ErrStudentNISNExists", err)
	}

	duplicatePhone := input
	duplicatePhone.StudentID = input.StudentID + "-phone"
	duplicatePhone.StudentUserID = input.StudentUserID + "-phone"
	duplicatePhone.ParentUserID = input.ParentUserID + "-phone"
	duplicatePhone.NISN = fmt.Sprintf("%010d", (suffix+1)%10_000_000_000)
	if err := service.CreateStudentWithAccounts(ctx, duplicatePhone); !errors.Is(err, ErrParentPhoneExists) {
		t.Fatalf("duplicate phone = %v, want ErrParentPhoneExists", err)
	}

	if err := service.DeleteStudentOnboarding(ctx, input.StudentUserID, input.ParentUserID); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if user, _ := service.UserByIdentifier(ctx, nisn); user != nil {
		t.Fatal("student account must be removed by cleanup")
	}
}
