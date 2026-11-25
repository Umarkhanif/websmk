package bkk

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	jobs []Job
	apps []Application
}

func (f fakeRepo) BkkJobs(context.Context) ([]Job, error) { return f.jobs, nil }
func (f fakeRepo) BkkApplications(context.Context) ([]Application, error) {
	return f.apps, nil
}
func (f fakeRepo) BkkApplicationsForStudent(context.Context, string) ([]Application, error) {
	return f.apps, nil
}

func TestStudentApplicationScope(t *testing.T) {
	svc := NewService(fakeRepo{apps: []Application{{AppID: "bkk-app-1"}}})
	if _, err := svc.ApplicationsForStudent(context.Background(), Viewer{Role: "student", StudentID: "siswa-1"}, "siswa-2"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("expected denied, got %v", err)
	}
	apps, err := svc.ApplicationsForStudent(context.Background(), Viewer{Role: "student", StudentID: "siswa-1"}, "siswa-1")
	if err != nil || len(apps) != 1 {
		t.Fatalf("apps %v, err %v", apps, err)
	}
}

func TestParentCannotReadListings(t *testing.T) {
	svc := NewService(fakeRepo{})
	if _, err := svc.Jobs(context.Background(), Viewer{Role: "parent", StudentID: "siswa-1"}); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("expected denied, got %v", err)
	}
}
