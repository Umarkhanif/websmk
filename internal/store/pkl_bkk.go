package store

import (
	"context"

	"wesmk/internal/domain/bkk"
	"wesmk/internal/domain/pkl"
)

const pklPlacementSelect = `
	SELECT
		COALESCE(p.app_id, ''), COALESCE(s.app_id, ''), p.company_name,
		p.industry_sector, p.address, p.gps_lat::float8, p.gps_lng::float8,
		p.radius_meters, p.supervisor_name, p.supervisor_phone,
		COALESCE(p.supervisor_pin_hash, '')
	FROM pkl_placements p
	JOIN students s ON s.id = p.student_id`

func (p *Postgres) PklPlacements(ctx context.Context) ([]pkl.Placement, error) {
	rows, err := p.pool.Query(ctx, pklPlacementSelect+` ORDER BY p.app_id NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklPlacements(rows)
}

func (p *Postgres) PklPlacementsForStudent(ctx context.Context, studentAppID string) ([]pkl.Placement, error) {
	rows, err := p.pool.Query(ctx, pklPlacementSelect+`
		WHERE s.app_id = $1
		ORDER BY p.app_id NULLS LAST`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklPlacements(rows)
}

func scanPklPlacements(rows invoiceRows) ([]pkl.Placement, error) {
	placements := make([]pkl.Placement, 0)
	for rows.Next() {
		var placement pkl.Placement
		if err := rows.Scan(
			&placement.AppID,
			&placement.StudentAppID,
			&placement.CompanyName,
			&placement.IndustrySector,
			&placement.Address,
			&placement.GpsLat,
			&placement.GpsLng,
			&placement.RadiusMeters,
			&placement.SupervisorName,
			&placement.SupervisorPhone,
			&placement.SupervisorPin,
		); err != nil {
			return nil, err
		}
		placements = append(placements, placement)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return placements, nil
}

const pklLogSelect = `
	SELECT
		COALESCE(l.app_id, ''), COALESCE(s.app_id, ''), COALESCE(s.full_name, ''),
		COALESCE(p.app_id, ''), p.company_name,
		COALESCE(to_char(COALESCE(l.checkin_time, l.created_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
		l.distance_meters, l.is_within_radius, l.selfie_url
	FROM pkl_attendance_logs l
	JOIN pkl_placements p ON p.id = l.placement_id
	JOIN students s ON s.id = p.student_id`

func (p *Postgres) PklLogs(ctx context.Context) ([]pkl.Log, error) {
	rows, err := p.pool.Query(ctx, pklLogSelect+` ORDER BY COALESCE(l.checkin_time, l.created_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklLogs(rows)
}

func (p *Postgres) PklLogsForStudent(ctx context.Context, studentAppID string) ([]pkl.Log, error) {
	rows, err := p.pool.Query(ctx, pklLogSelect+`
		WHERE s.app_id = $1
		ORDER BY COALESCE(l.checkin_time, l.created_at) DESC`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklLogs(rows)
}

func scanPklLogs(rows invoiceRows) ([]pkl.Log, error) {
	logs := make([]pkl.Log, 0)
	for rows.Next() {
		var log pkl.Log
		if err := rows.Scan(
			&log.AppID,
			&log.StudentAppID,
			&log.StudentName,
			&log.PlacementAppID,
			&log.CompanyName,
			&log.CheckinTime,
			&log.DistanceMeters,
			&log.IsWithinRadius,
			&log.SelfieURL,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

const pklJournalSelect = `
	SELECT
		COALESCE(j.app_id, ''), COALESCE(s.app_id, ''), COALESCE(s.full_name, ''),
		COALESCE(p.app_id, ''), p.company_name, to_char(j.journal_date, 'YYYY-MM-DD'),
		j.activity_title, j.competency_learned, j.work_photo_url,
		j.supervisor_approved, j.teacher_approved
	FROM pkl_daily_journals j
	JOIN pkl_placements p ON p.id = j.placement_id
	JOIN students s ON s.id = p.student_id`

func (p *Postgres) PklJournals(ctx context.Context) ([]pkl.Journal, error) {
	rows, err := p.pool.Query(ctx, pklJournalSelect+` ORDER BY j.journal_date DESC, j.app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklJournals(rows)
}

func (p *Postgres) PklJournalsForStudent(ctx context.Context, studentAppID string) ([]pkl.Journal, error) {
	rows, err := p.pool.Query(ctx, pklJournalSelect+`
		WHERE s.app_id = $1
		ORDER BY j.journal_date DESC, j.app_id`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPklJournals(rows)
}

func scanPklJournals(rows invoiceRows) ([]pkl.Journal, error) {
	journals := make([]pkl.Journal, 0)
	for rows.Next() {
		var journal pkl.Journal
		if err := rows.Scan(
			&journal.AppID,
			&journal.StudentAppID,
			&journal.StudentName,
			&journal.PlacementAppID,
			&journal.CompanyName,
			&journal.Date,
			&journal.ActivityTitle,
			&journal.CompetencyLearned,
			&journal.WorkPhotoURL,
			&journal.SupervisorApproved,
			&journal.TeacherApproved,
		); err != nil {
			return nil, err
		}
		journals = append(journals, journal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return journals, nil
}

func (p *Postgres) BkkJobs(ctx context.Context) ([]bkk.Job, error) {
	const query = `
		SELECT COALESCE(app_id, ''), company_name, job_title, majors, salary_range,
			COALESCE(to_char(deadline_date, 'YYYY-MM-DD'), ''), requirements, applicant_count
		FROM bkk_job_postings
		ORDER BY COALESCE(deadline_date, CURRENT_DATE) DESC, app_id NULLS LAST`
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]bkk.Job, 0)
	for rows.Next() {
		var job bkk.Job
		if err := rows.Scan(
			&job.AppID,
			&job.CompanyName,
			&job.JobTitle,
			&job.TargetedMajors,
			&job.SalaryRange,
			&job.DeadlineDate,
			&job.Requirements,
			&job.ApplicantCount,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

const bkkApplicationSelect = `
	SELECT
		COALESCE(a.app_id, ''), COALESCE(s.app_id, ''), COALESCE(s.full_name, ''),
		COALESCE(j.app_id, ''), j.job_title, j.company_name,
		COALESCE(to_char(a.applied_date, 'YYYY-MM-DD'), ''), a.status
	FROM bkk_job_applications a
	JOIN bkk_job_postings j ON j.id = a.posting_id
	JOIN students s ON s.id = a.student_id`

func (p *Postgres) BkkApplications(ctx context.Context) ([]bkk.Application, error) {
	rows, err := p.pool.Query(ctx, bkkApplicationSelect+` ORDER BY a.applied_date DESC NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBkkApplications(rows)
}

func (p *Postgres) BkkApplicationsForStudent(ctx context.Context, studentAppID string) ([]bkk.Application, error) {
	rows, err := p.pool.Query(ctx, bkkApplicationSelect+`
		WHERE s.app_id = $1
		ORDER BY a.applied_date DESC NULLS LAST`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBkkApplications(rows)
}

func scanBkkApplications(rows invoiceRows) ([]bkk.Application, error) {
	apps := make([]bkk.Application, 0)
	for rows.Next() {
		var app bkk.Application
		if err := rows.Scan(
			&app.AppID,
			&app.StudentAppID,
			&app.StudentName,
			&app.JobAppID,
			&app.JobTitle,
			&app.CompanyName,
			&app.AppliedDate,
			&app.Status,
		); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return apps, nil
}
