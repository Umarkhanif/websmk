package store

import (
	"context"

	"wesmk/internal/domain/attendance"
)

const attendanceSelect = `
	SELECT
		a.app_id, COALESCE(s.app_id, ''), a.student_name, COALESCE(s.nisn, ''),
		a.schedule_id, COALESCE(a.classroom_id::text, COALESCE(s.classroom_id::text, '')),
		COALESCE(to_char(a.attendance_date, 'YYYY-MM-DD'), ''),
		a.status, a.verified_source,
		COALESCE(to_char(a.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
	FROM attendances a
	JOIN students s ON s.id = a.student_id`

func (p *Postgres) AttendanceForStudent(ctx context.Context, studentAppID string) ([]attendance.Record, error) {
	rows, err := p.pool.Query(ctx, attendanceSelect+`
		WHERE s.app_id = $1
		ORDER BY a.updated_at DESC`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAttendance(rows)
}

func (p *Postgres) AllAttendances(ctx context.Context) ([]attendance.Record, error) {
	rows, err := p.pool.Query(ctx, attendanceSelect+`
		ORDER BY a.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAttendance(rows)
}

func scanAttendance(rows invoiceRows) ([]attendance.Record, error) {
	records := make([]attendance.Record, 0)
	for rows.Next() {
		var record attendance.Record
		if err := rows.Scan(
			&record.AppID,
			&record.StudentAppID,
			&record.StudentName,
			&record.NISN,
			&record.ScheduleID,
			&record.ClassroomID,
			&record.AttendanceDate,
			&record.Status,
			&record.VerifiedSource,
			&record.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}
