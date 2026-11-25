// Package pkl owns PKL placement, attendance, and journal reads.
package pkl

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrNotFound = errors.New("data PKL tidak ditemukan")

type Placement struct {
	AppID           string
	StudentAppID    string
	CompanyName     string
	IndustrySector  string
	Address         string
	GpsLat          float64
	GpsLng          float64
	RadiusMeters    int
	SupervisorName  string
	SupervisorPhone string
	SupervisorPin   string
}

// WithoutPin removes the partner credential before a portal receives the row.
func (p Placement) WithoutPin() Placement {
	p.SupervisorPin = ""
	return p
}

type Log struct {
	AppID          string
	StudentAppID   string
	StudentName    string
	PlacementAppID string
	CompanyName    string
	CheckinTime    string
	DistanceMeters int
	IsWithinRadius bool
	SelfieURL      string
}

type Journal struct {
	AppID              string
	StudentAppID       string
	StudentName        string
	PlacementAppID     string
	CompanyName        string
	Date               string
	ActivityTitle      string
	CompetencyLearned  string
	WorkPhotoURL       string
	SupervisorApproved bool
	TeacherApproved    bool
}

type Repository interface {
	PklPlacements(ctx context.Context) ([]Placement, error)
	PklPlacementsForStudent(ctx context.Context, studentAppID string) ([]Placement, error)
	PklLogs(ctx context.Context) ([]Log, error)
	PklLogsForStudent(ctx context.Context, studentAppID string) ([]Log, error)
	PklJournals(ctx context.Context) ([]Journal, error)
	PklJournalsForStudent(ctx context.Context, studentAppID string) ([]Journal, error)
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

func (s *Service) PlacementsForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Placement, error) {
	if err := s.authorizeStudent(viewer, studentAppID, true); err != nil {
		return nil, err
	}
	rows, err := s.repo.PklPlacementsForStudent(ctx, studentAppID)
	if err != nil {
		return nil, err
	}
	return withoutPins(rows), nil
}

func (s *Service) LogsForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Log, error) {
	if err := s.authorizeStudent(viewer, studentAppID, true); err != nil {
		return nil, err
	}
	return s.repo.PklLogsForStudent(ctx, studentAppID)
}

func (s *Service) JournalsForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Journal, error) {
	if err := s.authorizeStudent(viewer, studentAppID, true); err != nil {
		return nil, err
	}
	return s.repo.PklJournalsForStudent(ctx, studentAppID)
}

func (s *Service) AllPlacements(ctx context.Context, viewer Viewer) ([]Placement, error) {
	if err := s.authorizeStaff(viewer); err != nil {
		return nil, err
	}
	rows, err := s.repo.PklPlacements(ctx)
	if err != nil {
		return nil, err
	}
	return withoutPins(rows), nil
}

func (s *Service) AllLogs(ctx context.Context, viewer Viewer) ([]Log, error) {
	if err := s.authorizeStaff(viewer); err != nil {
		return nil, err
	}
	return s.repo.PklLogs(ctx)
}

func (s *Service) AllJournals(ctx context.Context, viewer Viewer) ([]Journal, error) {
	if err := s.authorizeStaff(viewer); err != nil {
		return nil, err
	}
	return s.repo.PklJournals(ctx)
}

func (s *Service) authorizeStudent(viewer Viewer, studentAppID string, own bool) error {
	if s == nil || s.repo == nil || studentAppID == "" {
		return ErrNotFound
	}
	isOwn := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourcePkl, authz.Ownership{Own: isOwn && own}) {
		return authz.ErrDenied
	}
	return nil
}

func (s *Service) authorizeStaff(viewer Viewer) error {
	if s == nil || s.repo == nil {
		return ErrNotFound
	}
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourcePkl, authz.Ownership{}) {
		return authz.ErrDenied
	}
	return nil
}

func withoutPins(rows []Placement) []Placement {
	out := make([]Placement, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.WithoutPin())
	}
	return out
}
