// Package bkk owns job-posting and application reads for BKK.
package bkk

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrNotFound = errors.New("data BKK tidak ditemukan")

type Job struct {
	AppID          string
	CompanyName    string
	JobTitle       string
	TargetedMajors []string
	SalaryRange    string
	DeadlineDate   string
	Requirements   []string
	ApplicantCount int
}

type Application struct {
	AppID        string
	StudentAppID string
	StudentName  string
	JobAppID     string
	JobTitle     string
	CompanyName  string
	AppliedDate  string
	Status       string
}

type Repository interface {
	BkkJobs(ctx context.Context) ([]Job, error)
	BkkApplications(ctx context.Context) ([]Application, error)
	BkkApplicationsForStudent(ctx context.Context, studentAppID string) ([]Application, error)
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

func (s *Service) Jobs(ctx context.Context, viewer Viewer) ([]Job, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotFound
	}
	own := authz.Ownership{Own: viewer.StudentID != ""}
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceBkk, own) {
		return nil, authz.ErrDenied
	}
	return s.repo.BkkJobs(ctx)
}

func (s *Service) ApplicationsForStudent(ctx context.Context, viewer Viewer, studentAppID string) ([]Application, error) {
	if s == nil || s.repo == nil || studentAppID == "" {
		return nil, ErrNotFound
	}
	own := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceBkk, authz.Ownership{Own: own}) {
		return nil, authz.ErrDenied
	}
	return s.repo.BkkApplicationsForStudent(ctx, studentAppID)
}

func (s *Service) AllApplications(ctx context.Context, viewer Viewer) ([]Application, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNotFound
	}
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceBkk, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.BkkApplications(ctx)
}
