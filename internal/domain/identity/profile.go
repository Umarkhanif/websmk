// Package identity owns student identity/profile reads. It is the first domain
// slice cut over from the app_state blob to the normalized tables.
package identity

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrProfileNotFound = errors.New("profil siswa tidak ditemukan")

type StudentProfile struct {
	AppID           string
	UserID          string
	NISN            string
	NIK             string
	FullName        string
	ClassroomID     string
	ClassroomName   string
	MajorCode       string
	ParentName      string
	ParentPhone     string
	IsKetuaKelas    bool
	IsSekretaris    bool
	IsAlumni        bool
	ScholarshipType string
	DiscountPercent float64
	AttitudeScore   int
	BMWChoice       string
	HomeroomNote    string
}

// Repository is implemented by internal/store. It resolves the app-level
// student ID (the ID stored in users.student_profile_id) to a domain row.
type Repository interface {
	StudentProfile(ctx context.Context, studentAppID string) (*StudentProfile, error)
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

// ForViewer enforces the profile policy before returning any row, so a handler
// cannot accidentally hand another family's profile to a student or parent.
func (s *Service) ForViewer(ctx context.Context, viewer Viewer, studentID string) (*StudentProfile, error) {
	if s == nil || s.repo == nil {
		return nil, ErrProfileNotFound
	}
	if studentID == "" {
		return nil, ErrProfileNotFound
	}
	profile, err := s.repo.StudentProfile(ctx, studentID)
	if err != nil {
		return nil, err
	}
	own := (viewer.StudentID != "" && viewer.StudentID == profile.AppID) ||
		(viewer.UserID != "" && viewer.UserID == profile.UserID)
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceStudentProfile, authz.Ownership{Own: own}) {
		return nil, authz.ErrDenied
	}
	return profile, nil
}
