package attendance

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	records []Record
}

func (f fakeRepo) AttendanceForStudent(context.Context, string) ([]Record, error) {
	return f.records, nil
}
func (f fakeRepo) AllAttendances(context.Context) ([]Record, error) { return f.records, nil }

func TestAttendanceViewerPolicy(t *testing.T) {
	service := NewService(fakeRepo{records: []Record{{AppID: "att-1"}}})

	if _, err := service.ForViewer(context.Background(), Viewer{Role: "student", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("student own attendance: %v", err)
	}
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "parent", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("parent own attendance: %v", err)
	}
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "student", StudentID: "std-2"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("student foreign attendance err = %v, want denied", err)
	}
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "treasurer"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("treasurer attendance err = %v, want denied", err)
	}
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "student"}, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty student err = %v, want not found", err)
	}
}

func TestAttendanceStaffPolicy(t *testing.T) {
	service := NewService(fakeRepo{})
	for _, role := range []string{"teacher", "piket", "discipline", "admin_tu", "headmaster"} {
		if _, err := service.AllForViewer(context.Background(), Viewer{Role: role}); err != nil {
			t.Fatalf("%s all attendance: %v", role, err)
		}
	}
	for _, role := range []string{"student", "parent", "treasurer", "developer", ""} {
		if _, err := service.AllForViewer(context.Background(), Viewer{Role: role}); !errors.Is(err, authz.ErrDenied) {
			t.Fatalf("%s all attendance err = %v, want denied", role, err)
		}
	}
}
