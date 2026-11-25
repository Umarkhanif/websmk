package pkl

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	placements []Placement
	logs       []Log
	journals   []Journal
}

func (f fakeRepo) PklPlacements(context.Context) ([]Placement, error) { return f.placements, nil }
func (f fakeRepo) PklPlacementsForStudent(context.Context, string) ([]Placement, error) {
	return f.placements, nil
}
func (f fakeRepo) PklLogs(context.Context) ([]Log, error) { return f.logs, nil }
func (f fakeRepo) PklLogsForStudent(context.Context, string) ([]Log, error) {
	return f.logs, nil
}
func (f fakeRepo) PklJournals(context.Context) ([]Journal, error) { return f.journals, nil }
func (f fakeRepo) PklJournalsForStudent(context.Context, string) ([]Journal, error) {
	return f.journals, nil
}

func TestStudentPlacementRedactsPin(t *testing.T) {
	svc := NewService(fakeRepo{placements: []Placement{{AppID: "pkl-1", SupervisorPin: "secret"}}})
	rows, err := svc.PlacementsForStudent(context.Background(), Viewer{Role: "student", StudentID: "siswa-1"}, "siswa-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].SupervisorPin != "" {
		t.Fatalf("pin leaked to student: %q", rows[0].SupervisorPin)
	}
}

func TestStudentCannotReadAnotherStudent(t *testing.T) {
	svc := NewService(fakeRepo{})
	_, err := svc.LogsForStudent(context.Background(), Viewer{Role: "student", StudentID: "siswa-1"}, "siswa-2")
	if !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("expected denied, got %v", err)
	}
}

func TestTeacherReadsAllButNoPin(t *testing.T) {
	svc := NewService(fakeRepo{placements: []Placement{{AppID: "pkl-1", SupervisorPin: "secret"}}})
	rows, err := svc.AllPlacements(context.Background(), Viewer{Role: "teacher"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].SupervisorPin != "" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}
