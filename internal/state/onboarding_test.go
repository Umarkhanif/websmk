package state

import (
	"testing"
	"time"
)

func TestAddNewStudentUsesRealIdentityAndRejectsDuplicateNISN(t *testing.T) {
	doc := map[string]any{}
	input := NewStudentInput{
		ID: "std-test", UserID: "usr-std-test", ParentUserID: "usr-prt-test",
		FullName: "Siswa Uji", NISN: "0098765432", ClassroomID: "11111111-1111-1111-1111-111111111111", Classroom: "X TKJ 1", MajorCode: "TKJ",
		ParentName: "Wali Uji", ParentPhone: "081234567890",
	}
	now := time.Date(2026, time.November, 23, 10, 0, 0, 0, time.UTC)
	if err := AddNewStudent(doc, input, now); err != nil {
		t.Fatalf("add student: %v", err)
	}
	student := asMap(asList(doc["students"])[0])
	if text(student["userId"]) != input.UserID || text(student["parentName"]) != input.ParentName || text(student["classroomId"]) != input.ClassroomID {
		t.Fatalf("student snapshot = %+v", student)
	}
	if text(student["parentPhone"]) != input.ParentPhone {
		t.Fatalf("parent phone = %q", text(student["parentPhone"]))
	}
	invoice := asMap(asList(doc["invoices"])[0])
	if text(invoice["periodMonthYear"]) != "November 2026" {
		t.Fatalf("invoice period = %q", text(invoice["periodMonthYear"]))
	}
	if text(invoice["vaNumber"]) != "" {
		t.Fatalf("VA number should stay empty until a gateway issues one: %q", text(invoice["vaNumber"]))
	}
	if err := AddNewStudent(doc, input, now); err == nil {
		t.Fatal("duplicate NISN must be rejected")
	}
}
