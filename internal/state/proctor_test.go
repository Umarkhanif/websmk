package state

import (
	"errors"
	"testing"
	"time"
)

const proctorDoc = `{
  "students": [{"id": "std-1", "nisn": "0058471290"}],
  "proctorStudents": [{"id": "ps-1", "studentId": "std-1", "status": "ACTIVE", "warnCount": 2, "lockCount": 0}]
}`

func TestUpdateProctorStatusRejectsClientUnlock(t *testing.T) {
	doc := mustDoc(t, proctorDoc)
	err := UpdateProctorStatus(doc, "std-1", "ACTIVE", "client says ok", time.Now())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
	ps := asMap(asList(doc["proctorStudents"])[0])
	if status := text(ps["status"]); status != "ACTIVE" {
		t.Fatalf("status = %q, want unchanged ACTIVE", status)
	}
}

func TestUpdateProctorStatusLockIsPermanent(t *testing.T) {
	doc := mustDoc(t, proctorDoc)
	now := time.Now()

	if err := UpdateProctorStatus(doc, "std-1", "WARN", "blur", now); err != nil {
		t.Fatalf("warn: %v", err)
	}
	ps := asMap(asList(doc["proctorStudents"])[0])
	if status := text(ps["status"]); status != "LOCKED" {
		t.Fatalf("status after 3rd warn = %q, want LOCKED", status)
	}
	warn, _ := number(ps["warnCount"])
	if warn != 3 {
		t.Fatalf("warnCount = %v, want 3", warn)
	}
	if token := text(ps["unlockToken"]); token == "" {
		t.Fatal("server must issue an unlock token on lock")
	}

	before, _ := number(ps["warnCount"])
	if err := UpdateProctorStatus(doc, "std-1", "WARN", "again", now.Add(time.Second)); err != nil {
		t.Fatalf("warn after lock: %v", err)
	}
	after, _ := number(asMap(asList(doc["proctorStudents"])[0])["warnCount"])
	if after != before {
		t.Fatalf("warnCount changed after lock: %v -> %v", before, after)
	}

	// Only the teacher/QR unlock path may restore ACTIVE.
	UnlockProctorStudent(doc, "std-1")
	if status := text(asMap(asList(doc["proctorStudents"])[0])["status"]); status != "ACTIVE" {
		t.Fatalf("status after authorized unlock = %q, want ACTIVE", status)
	}
}
