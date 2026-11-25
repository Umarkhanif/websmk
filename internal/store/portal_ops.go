package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrStudentNotFound = errors.New("siswa tidak ditemukan")

type ClassRosterEntry struct {
	StudentAppID   string
	FullName       string
	NISN           string
	AttendanceID   string
	Status         string
	VerifiedSource string
	GateTapTime    string
}

type ParentLeaveEntry struct {
	ID        string
	LeaveType string
	LeaveDate string
	Note      string
	Status    string
	CreatedAt string
}

type GateLogEntry struct {
	ID          string
	NISN        string
	StudentName string
	LogType     string
	Reason      string
	PassCode    string
	RecordedAt  string
}

type SubstituteAssignmentEntry struct {
	ID                string
	ScheduleID        string
	ClassName         string
	SubjectID         string
	TeacherID         string
	TeacherName       string
	OriginalTeacherID string
	OriginalTeacher   string
	Note              string
	AssignmentDate    string
}

type ClassJournalEntry struct {
	ID          string
	ScheduleID  string
	ClassName   string
	SubjectID   string
	TeacherName string
	TeachDate   string
	Topic       string
	Notes       string
	SmartNotes  string
	UpdatedAt   string
}

type StudentDirectoryEntry struct {
	ID            string
	NISN          string
	FullName      string
	ClassroomName string
	MajorCode     string
	ParentName    string
	ParentPhone   string
	IsAlumni      bool
}

// ClassroomRoster returns the enrolled students and their latest attendance
// record for one class/date. Students without a record stay in the roster so a
// teacher can mark them present instead of losing them from the list.
func (p *Postgres) ClassroomRoster(ctx context.Context, classroomID string, date time.Time) ([]ClassRosterEntry, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT s.app_id, COALESCE(s.full_name, ''), COALESCE(s.nisn, ''),
		       COALESCE(a.app_id, ''), COALESCE(a.status, ''), COALESCE(a.verified_source, ''),
		       COALESCE(to_char(a.updated_at AT TIME ZONE 'Asia/Jakarta', 'HH24:MI'), '')
		FROM students s
		LEFT JOIN LATERAL (
			SELECT a.app_id, a.status, a.verified_source, a.updated_at
			FROM attendances a
			WHERE a.student_id = s.id AND a.attendance_date = $2
			ORDER BY a.updated_at DESC
			LIMIT 1
		) a ON TRUE
		WHERE s.classroom_id = $1::uuid AND s.is_alumni = FALSE
		ORDER BY s.full_name
	`, classroomID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]ClassRosterEntry, 0)
	for rows.Next() {
		var entry ClassRosterEntry
		if err := rows.Scan(&entry.StudentAppID, &entry.FullName, &entry.NISN,
			&entry.AttendanceID, &entry.Status, &entry.VerifiedSource, &entry.GateTapTime); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// TeacherCanManageClassroom permits a teacher only for their homeroom or a
// class they teach on that weekday.
func (p *Postgres) TeacherCanManageClassroom(ctx context.Context, teacherID, classroomID string, date time.Time) (bool, error) {
	var allowed bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM classrooms
			WHERE id = $2::uuid AND homeroom_teacher_id = $1
		) OR EXISTS (
			SELECT 1 FROM schedules
			WHERE classroom_id = $2::uuid AND teacher_id = $1 AND day_of_week = $3
		)
	`, teacherID, classroomID, schoolWeekday(date)).Scan(&allowed)
	return allowed, err
}

func (p *Postgres) AttendanceClassroom(ctx context.Context, attendanceID string) (string, time.Time, error) {
	var classroomID string
	var date time.Time
	err := p.pool.QueryRow(ctx, `
		SELECT COALESCE(classroom_id::text, ''), attendance_date
		FROM attendances WHERE app_id = $1
	`, attendanceID).Scan(&classroomID, &date)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("attendance not found: %w", err)
	}
	return classroomID, date, nil
}

// UpsertStudentAttendance updates the latest record for the date, or creates
// the first one. The app id is deterministic so repeated parent submissions do
// not multiply rows.
func (p *Postgres) UpsertStudentAttendance(ctx context.Context, studentAppID string, date time.Time, status, source string) (string, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var appID string
	err = tx.QueryRow(ctx, `
		WITH latest AS (
			SELECT a.id
			FROM attendances a
			JOIN students s ON s.id = a.student_id
			WHERE s.app_id = $1 AND a.attendance_date = $2
			ORDER BY a.updated_at DESC
			LIMIT 1
		)
		UPDATE attendances a
		SET status = $3, verified_source = $4, updated_at = now()
		FROM latest
		WHERE a.id = latest.id
		RETURNING a.app_id
	`, studentAppID, date, status, source).Scan(&appID)
	if err == nil {
		return appID, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO attendances (
			app_id, student_id, schedule_id, student_name, status, verified_source,
			attendance_date, classroom_id, updated_at
		)
		SELECT 'leave-' || s.app_id || '-' || $2::date, s.id, '', COALESCE(s.full_name, ''),
		       $3, $4, $2, s.classroom_id, now()
		FROM students s
		WHERE s.app_id = $1
		RETURNING app_id
	`, studentAppID, date, status, source).Scan(&appID)
	if err != nil {
		return "", err
	}
	return appID, tx.Commit(ctx)
}

func (p *Postgres) CreateParentLeaveRequest(ctx context.Context, studentAppID, leaveType string, date time.Time, note string) (ParentLeaveEntry, error) {
	leaveType = strings.ToUpper(strings.TrimSpace(leaveType))
	if leaveType != "SAKIT" && leaveType != "IZIN" {
		return ParentLeaveEntry{}, errors.New("jenis izin tidak valid")
	}
	status := "IZIN_ORTU"
	if leaveType == "SAKIT" {
		status = "SAKIT_ORTU"
	}
	var entry ParentLeaveEntry
	err := p.pool.QueryRow(ctx, `
		INSERT INTO parent_leave_requests (student_id, leave_type, leave_date, note, status)
		SELECT s.id, $2, $3, $4, 'TERCATAT'
		FROM students s WHERE s.app_id = $1
		RETURNING id::text, leave_type, to_char(leave_date, 'YYYY-MM-DD'), note, status,
		          to_char(created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI')
	`, studentAppID, leaveType, date, strings.TrimSpace(note)).Scan(
		&entry.ID, &entry.LeaveType, &entry.LeaveDate, &entry.Note, &entry.Status, &entry.CreatedAt)
	if err != nil {
		return ParentLeaveEntry{}, ErrStudentNotFound
	}
	source := fmt.Sprintf("Izin orang tua (%s): %s", entry.LeaveDate, strings.TrimSpace(note))
	if _, err := p.UpsertStudentAttendance(ctx, studentAppID, date, status, source); err != nil {
		return ParentLeaveEntry{}, err
	}
	return entry, nil
}

func (p *Postgres) ParentLeavesForStudent(ctx context.Context, studentAppID string) ([]ParentLeaveEntry, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT l.id::text, l.leave_type, to_char(l.leave_date, 'YYYY-MM-DD'), l.note, l.status,
		       to_char(l.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI')
		FROM parent_leave_requests l
		JOIN students s ON s.id = l.student_id
		WHERE s.app_id = $1
		ORDER BY l.leave_date DESC, l.created_at DESC
		LIMIT 30
	`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]ParentLeaveEntry, 0)
	for rows.Next() {
		var entry ParentLeaveEntry
		if err := rows.Scan(&entry.ID, &entry.LeaveType, &entry.LeaveDate, &entry.Note,
			&entry.Status, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (p *Postgres) RecentGateLogs(ctx context.Context, limit int) ([]GateLogEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, nisn, student_name, log_type, reason, pass_code,
		       to_char(recorded_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI')
		FROM gate_logs ORDER BY recorded_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]GateLogEntry, 0)
	for rows.Next() {
		var entry GateLogEntry
		if err := rows.Scan(&entry.ID, &entry.NISN, &entry.StudentName, &entry.LogType,
			&entry.Reason, &entry.PassCode, &entry.RecordedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (p *Postgres) CreateGateLog(ctx context.Context, nisn, logType, reason, passCode, createdBy string) (GateLogEntry, error) {
	nisn = strings.TrimSpace(nisn)
	logType = strings.TrimSpace(logType)
	if nisn == "" || logType == "" {
		return GateLogEntry{}, errors.New("NISN dan jenis log wajib diisi")
	}
	var entry GateLogEntry
	err := p.pool.QueryRow(ctx, `
		INSERT INTO gate_logs (student_id, nisn, student_name, log_type, reason, pass_code, created_by_user_id)
		SELECT s.id, s.nisn, COALESCE(s.full_name, ''), $2, $3, $4, NULLIF($5, '')
		FROM students s WHERE s.nisn = $1
		RETURNING id::text, nisn, student_name, log_type, reason, pass_code,
		          to_char(recorded_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI')
	`, nisn, logType, strings.TrimSpace(reason), passCode, createdBy).Scan(
		&entry.ID, &entry.NISN, &entry.StudentName, &entry.LogType, &entry.Reason, &entry.PassCode, &entry.RecordedAt)
	if err != nil {
		return GateLogEntry{}, ErrStudentNotFound
	}
	return entry, nil
}

func (p *Postgres) SubstituteAssignmentsForDate(ctx context.Context, date time.Time) ([]SubstituteAssignmentEntry, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT sa.id::text, COALESCE(sa.schedule_id::text, ''), COALESCE(c.name, ''),
		       COALESCE(sc.subject_id, ''), COALESCE(tr.id, ''), COALESCE(tr.full_name, ''),
		       COALESCE(orig.id, ''), COALESCE(orig.full_name, ''), sa.note,
		       to_char(sa.assignment_date, 'YYYY-MM-DD')
		FROM substitute_assignments sa
		LEFT JOIN schedules sc ON sc.id = sa.schedule_id
		LEFT JOIN classrooms c ON c.id = sc.classroom_id
		LEFT JOIN users tr ON tr.id = sa.teacher_id
		LEFT JOIN users orig ON orig.id = COALESCE(sa.original_teacher_id, sc.teacher_id)
		WHERE sa.assignment_date = $1
		ORDER BY sa.assigned_at DESC
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]SubstituteAssignmentEntry, 0)
	for rows.Next() {
		var entry SubstituteAssignmentEntry
		if err := rows.Scan(&entry.ID, &entry.ScheduleID, &entry.ClassName, &entry.SubjectID,
			&entry.TeacherID, &entry.TeacherName, &entry.OriginalTeacherID, &entry.OriginalTeacher,
			&entry.Note, &entry.AssignmentDate); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (p *Postgres) CreateSubstituteAssignment(ctx context.Context, scheduleID, teacherID, note, assignedBy string, date time.Time) (SubstituteAssignmentEntry, error) {
	// One substitute per schedule slot per day. Reassigning replaces the
	// previous teacher instead of creating duplicate rows.
	if _, err := p.pool.Exec(ctx, `DELETE FROM substitute_assignments WHERE schedule_id = $1::uuid AND assignment_date = $2`, scheduleID, date); err != nil {
		return SubstituteAssignmentEntry{}, err
	}
	var entry SubstituteAssignmentEntry
	err := p.pool.QueryRow(ctx, `
		WITH schedule AS (
			SELECT sc.id, sc.classroom_id, sc.subject_id, sc.teacher_id
			FROM schedules sc WHERE sc.id = $1::uuid
		), inserted AS (
			INSERT INTO substitute_assignments (
				teacher_id, schedule_id, assignment_date, note, assigned_by_user_id, original_teacher_id
			)
			SELECT $2, schedule.id, $5::date, $3, NULLIF($4, ''), schedule.teacher_id
			FROM schedule
			RETURNING *
		)
		SELECT i.id::text, i.schedule_id::text, COALESCE(c.name, ''), COALESCE(sc.subject_id, ''),
		       COALESCE(tr.id, ''), COALESCE(tr.full_name, ''),
		       COALESCE(orig.id, ''), COALESCE(orig.full_name, ''), i.note,
		       to_char(i.assignment_date, 'YYYY-MM-DD')
		FROM inserted i
		JOIN schedules sc ON sc.id = i.schedule_id
		JOIN classrooms c ON c.id = sc.classroom_id
		LEFT JOIN users tr ON tr.id = i.teacher_id
		LEFT JOIN users orig ON orig.id = sc.teacher_id
	`, scheduleID, teacherID, strings.TrimSpace(note), assignedBy, date).Scan(
		&entry.ID, &entry.ScheduleID, &entry.ClassName, &entry.SubjectID,
		&entry.TeacherID, &entry.TeacherName, &entry.OriginalTeacherID, &entry.OriginalTeacher,
		&entry.Note, &entry.AssignmentDate)
	if err != nil {
		return SubstituteAssignmentEntry{}, fmt.Errorf("buat penugasan inval: %w", err)
	}
	return entry, nil
}

const journalSelect = `
	SELECT j.id::text, COALESCE(j.schedule_id::text, ''), COALESCE(c.name, ''),
	       COALESCE(NULLIF(j.subject_id, ''), COALESCE(sc.subject_id, '')), COALESCE(u.full_name, ''),
	       to_char(j.teach_date, 'YYYY-MM-DD'), j.syllabus_topic, j.teacher_notes,
	       j.ai_smart_notes,
	       COALESCE(to_char(j.updated_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI'), '')
	FROM class_journals j
	LEFT JOIN schedules sc ON sc.id = j.schedule_id
	LEFT JOIN classrooms c ON c.id = COALESCE(j.classroom_id, sc.classroom_id)
	LEFT JOIN users u ON u.id = COALESCE(j.teacher_id, sc.teacher_id)`

func (p *Postgres) ClassJournalForScheduleDate(ctx context.Context, scheduleID string, date time.Time) (ClassJournalEntry, bool) {
	var entry ClassJournalEntry
	err := p.pool.QueryRow(ctx, journalSelect+`
		WHERE j.schedule_id = $1::uuid AND j.teach_date = $2
		ORDER BY j.updated_at DESC LIMIT 1
	`, scheduleID, date).Scan(&entry.ID, &entry.ScheduleID, &entry.ClassName, &entry.SubjectID,
		&entry.TeacherName, &entry.TeachDate, &entry.Topic, &entry.Notes, &entry.SmartNotes, &entry.UpdatedAt)
	if err != nil {
		return ClassJournalEntry{}, false
	}
	return entry, true
}

func (p *Postgres) LatestClassJournalForClassroom(ctx context.Context, classroomID string) (ClassJournalEntry, bool) {
	var entry ClassJournalEntry
	err := p.pool.QueryRow(ctx, journalSelect+`
		WHERE COALESCE(j.classroom_id, sc.classroom_id) = $1::uuid
		ORDER BY j.teach_date DESC, j.updated_at DESC LIMIT 1
	`, classroomID).Scan(&entry.ID, &entry.ScheduleID, &entry.ClassName, &entry.SubjectID,
		&entry.TeacherName, &entry.TeachDate, &entry.Topic, &entry.Notes, &entry.SmartNotes, &entry.UpdatedAt)
	if err != nil {
		return ClassJournalEntry{}, false
	}
	return entry, true
}

func (p *Postgres) SaveClassJournal(ctx context.Context, scheduleID, teacherID string, date time.Time, topic, notes, smartNotes string) error {
	topic = strings.TrimSpace(topic)
	notes = strings.TrimSpace(notes)
	if topic == "" || notes == "" {
		return errors.New("topik dan catatan mengajar wajib diisi")
	}
	_, err := p.pool.Exec(ctx, `
		INSERT INTO class_journals (
			schedule_id, classroom_id, teacher_id, subject_id, teach_date,
			syllabus_topic, teacher_notes, ai_smart_notes
		)
		SELECT sc.id, sc.classroom_id, $2, sc.subject_id, $3, $4, $5, $6
		FROM schedules sc
		WHERE sc.id = $1::uuid AND sc.teacher_id = $2
		ON CONFLICT (schedule_id, teach_date) WHERE schedule_id IS NOT NULL
		DO UPDATE SET
			syllabus_topic = EXCLUDED.syllabus_topic,
			teacher_notes = EXCLUDED.teacher_notes,
			ai_smart_notes = EXCLUDED.ai_smart_notes,
			updated_at = now()
	`, scheduleID, teacherID, date, topic, notes, strings.TrimSpace(smartNotes))
	if err != nil {
		return fmt.Errorf("simpan jurnal: %w", err)
	}
	return nil
}

func (p *Postgres) StudentDirectory(ctx context.Context, query string, limit, offset int) ([]StudentDirectoryEntry, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	query = strings.TrimSpace(query)
	pattern := "%" + query + "%"
	var total int
	if err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM students s
		LEFT JOIN classrooms c ON c.id = s.classroom_id
		WHERE ($1 = '' OR s.full_name ILIKE $1 OR s.nisn ILIKE $1 OR c.name ILIKE $1)
	`, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := p.pool.Query(ctx, `
		SELECT s.app_id, COALESCE(s.nisn, ''), COALESCE(s.full_name, ''),
		       COALESCE(c.name, s.classroom_name, ''), COALESCE(s.major_code, c.major_code, ''),
		       COALESCE(s.parent_name, ''), COALESCE(s.parent_phone, ''), s.is_alumni
		FROM students s
		LEFT JOIN classrooms c ON c.id = s.classroom_id
		WHERE ($1 = '' OR s.full_name ILIKE $1 OR s.nisn ILIKE $1 OR c.name ILIKE $1)
		ORDER BY s.is_alumni, c.name, s.full_name
		LIMIT $2 OFFSET $3
	`, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	entries := make([]StudentDirectoryEntry, 0)
	for rows.Next() {
		var entry StudentDirectoryEntry
		if err := rows.Scan(&entry.ID, &entry.NISN, &entry.FullName, &entry.ClassroomName,
			&entry.MajorCode, &entry.ParentName, &entry.ParentPhone, &entry.IsAlumni); err != nil {
			return nil, 0, err
		}
		entries = append(entries, entry)
	}
	return entries, total, rows.Err()
}

func schoolWeekday(date time.Time) int {
	day := int(date.Weekday())
	if day == 0 {
		return 7
	}
	return day
}
