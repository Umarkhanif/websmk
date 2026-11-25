package httpapi

import (
	"context"
	"errors"
	"log"

	"wesmk/internal/authz"
	"wesmk/internal/domain/bkk"
	"wesmk/internal/domain/pkl"
	"wesmk/internal/model"
	"wesmk/internal/ui"
)

func domainPklViewer(user *model.User) pkl.Viewer {
	if user == nil {
		return pkl.Viewer{}
	}
	return pkl.Viewer{UserID: user.ID, Role: user.Role, StudentID: user.StudentProfileID}
}

func domainBkkViewer(user *model.User) bkk.Viewer {
	if user == nil {
		return bkk.Viewer{}
	}
	return bkk.Viewer{UserID: user.ID, Role: user.Role, StudentID: user.StudentProfileID}
}

func pklPlacementViewFromDomain(placement pkl.Placement) ui.PklPlacementView {
	return ui.PklPlacementView{
		ID:              placement.AppID,
		CompanyName:     placement.CompanyName,
		IndustrySector:  placement.IndustrySector,
		Address:         placement.Address,
		GpsLat:          placement.GpsLat,
		GpsLng:          placement.GpsLng,
		RadiusMeters:    placement.RadiusMeters,
		SupervisorName:  placement.SupervisorName,
		SupervisorPhone: placement.SupervisorPhone,
	}
}

func pklPlacementViewsFromDomain(placements []pkl.Placement) []ui.PklPlacementView {
	views := make([]ui.PklPlacementView, 0, len(placements))
	for _, placement := range placements {
		views = append(views, pklPlacementViewFromDomain(placement))
	}
	return views
}

func pklLogViewFromDomain(log pkl.Log) ui.PklLogView {
	geofence := "DI LUAR RADIUS"
	if log.IsWithinRadius {
		geofence = "DI DALAM RADIUS"
	}
	return ui.PklLogView{
		ID:             log.AppID,
		StudentID:      log.StudentAppID,
		StudentName:    log.StudentName,
		CompanyName:    log.CompanyName,
		Timestamp:      log.CheckinTime,
		DistanceMeters: log.DistanceMeters,
		GeofenceStatus: geofence,
		ExifVerified:   log.SelfieURL != "",
	}
}

func pklLogViewsFromDomain(logs []pkl.Log) []ui.PklLogView {
	views := make([]ui.PklLogView, 0, len(logs))
	for _, log := range logs {
		views = append(views, pklLogViewFromDomain(log))
	}
	return views
}

func pklJournalViewFromDomain(journal pkl.Journal) ui.PklJournalView {
	return ui.PklJournalView{
		ID:                 journal.AppID,
		StudentID:          journal.StudentAppID,
		StudentName:        journal.StudentName,
		PlacementID:        journal.PlacementAppID,
		CompanyName:        journal.CompanyName,
		Date:               journal.Date,
		ActivityTitle:      journal.ActivityTitle,
		CompetencyLearned:  journal.CompetencyLearned,
		SupervisorApproved: journal.SupervisorApproved,
		TeacherApproved:    journal.TeacherApproved,
	}
}

func pklJournalViewsFromDomain(journals []pkl.Journal) []ui.PklJournalView {
	views := make([]ui.PklJournalView, 0, len(journals))
	for _, journal := range journals {
		views = append(views, pklJournalViewFromDomain(journal))
	}
	return views
}

func bkkJobViewFromDomain(job bkk.Job) ui.BkkJobView {
	majors := ""
	for i, major := range job.TargetedMajors {
		if i > 0 {
			majors += ", "
		}
		majors += major
	}
	return ui.BkkJobView{
		ID:             job.AppID,
		CompanyName:    job.CompanyName,
		JobTitle:       job.JobTitle,
		TargetedMajors: majors,
		SalaryRange:    job.SalaryRange,
		DeadlineDate:   job.DeadlineDate,
		ApplicantCount: job.ApplicantCount,
	}
}

func bkkAppViewFromDomain(app bkk.Application) ui.BkkAppView {
	return ui.BkkAppView{
		ID:          app.AppID,
		JobID:       app.JobAppID,
		JobTitle:    app.JobTitle,
		CompanyName: app.CompanyName,
		StudentID:   app.StudentAppID,
		StudentName: app.StudentName,
		AppliedDate: app.AppliedDate,
		Status:      app.Status,
	}
}

func (s *Server) domainStudentPkl(ctx context.Context, user *model.User, studentID string) ([]pkl.Placement, []pkl.Log, []pkl.Journal, bool) {
	if !s.cfg.DomainPklBkk || s.pklSvc == nil || user == nil || studentID == "" {
		return nil, nil, nil, false
	}
	viewer := domainPklViewer(user)
	placements, err := s.pklSvc.PlacementsForStudent(ctx, viewer, studentID)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, nil, false
	}
	logs, err := s.pklSvc.LogsForStudent(ctx, viewer, studentID)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, nil, false
	}
	journals, err := s.pklSvc.JournalsForStudent(ctx, viewer, studentID)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, nil, false
	}
	return placements, logs, journals, true
}

func (s *Server) domainTeacherPkl(ctx context.Context, user *model.User) ([]pkl.Log, []pkl.Journal, bool) {
	if !s.cfg.DomainPklBkk || s.pklSvc == nil || user == nil {
		return nil, nil, false
	}
	logs, err := s.pklSvc.AllLogs(ctx, domainPklViewer(user))
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	journals, err := s.pklSvc.AllJournals(ctx, domainPklViewer(user))
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	return logs, journals, true
}

func (s *Server) domainAdminPkl(ctx context.Context, user *model.User) ([]pkl.Placement, bool) {
	if !s.cfg.DomainPklBkk || s.pklSvc == nil || user == nil {
		return nil, false
	}
	placements, err := s.pklSvc.AllPlacements(ctx, domainPklViewer(user))
	if err != nil {
		logPklBkkFallback(err)
		return nil, false
	}
	return placements, true
}

func (s *Server) domainStudentBkk(ctx context.Context, user *model.User, studentID string) ([]bkk.Job, []bkk.Application, bool) {
	if !s.cfg.DomainPklBkk || s.bkkSvc == nil || user == nil || studentID == "" {
		return nil, nil, false
	}
	viewer := domainBkkViewer(user)
	jobs, err := s.bkkSvc.Jobs(ctx, viewer)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	apps, err := s.bkkSvc.ApplicationsForStudent(ctx, viewer, studentID)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	return jobs, apps, true
}

func (s *Server) domainAdminBkk(ctx context.Context, user *model.User) ([]bkk.Job, []bkk.Application, bool) {
	if !s.cfg.DomainPklBkk || s.bkkSvc == nil || user == nil {
		return nil, nil, false
	}
	viewer := domainBkkViewer(user)
	jobs, err := s.bkkSvc.Jobs(ctx, viewer)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	apps, err := s.bkkSvc.AllApplications(ctx, viewer)
	if err != nil {
		logPklBkkFallback(err)
		return nil, nil, false
	}
	return jobs, apps, true
}

func logPklBkkFallback(err error) {
	if err == nil || errors.Is(err, authz.ErrDenied) || errors.Is(err, pkl.ErrNotFound) || errors.Is(err, bkk.ErrNotFound) {
		return
	}
	log.Printf("domain PKL/BKK read failed: %v", err)
}
