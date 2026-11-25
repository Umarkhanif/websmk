// Package attendance owns daily attendance reads from the normalized table.
package attendance

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrNotFound = errors.New("data presensi tidak ditemukan")

type Record struct {
	AppID          string
	StudentAppID   string
	StudentName    string
	NISN           string
	ScheduleID     string
	ClassroomID    string
	AttendanceDate string
	Status         string
	VerifiedSource string
	UpdatedAt      string
}

type Repository interface {
	AttendanceForStudent(ctx context.Context, studentAppID string) ([]Record, error)
	AllAttendances(ctx context.Context) ([]Record, error)
}

type Viewer struct {
	UserID    string
	Role      string
	StudentID string
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ForViewer(ctx context.Context, viewer Viewer, studentAppID string) ([]Record, error) {
	if s == nil || s.repo == nil || studentAppID == "" {
		return nil, ErrNotFound
	}
	own := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceAttendance, authz.Ownership{Own: own}) {
		return nil, authz.ErrDenied
	}
	return s.repo.AttendanceForStudent(ctx, studentAppID)
}

func (s *Service) AllForViewer(ctx context.Context, viewer Viewer) ([]Record, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotFound
	}
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceAttendance, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.AllAttendances(ctx)
}
