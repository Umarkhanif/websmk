package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrMasterDataExists = errors.New("data master dengan nama tersebut sudah ada")
	ErrMasterDataInUse  = errors.New("data master masih dipakai")
)

// AcademicYear is one semester of a school year. Exactly one row may be active.
type AcademicYear struct {
	ID        string
	Label     string
	Semester  string
	StartDate time.Time
	EndDate   time.Time
	Active    bool
}

type Classroom struct {
	ID              string
	Name            string
	GradeLevel      int
	MajorCode       string
	MajorName       string
	HomeroomTeacher string
	HomeroomName    string
	StudentCount    int
}

type Schedule struct {
	ID          string
	ClassroomID string
	ClassName   string
	SubjectID   string
	TeacherID   string
	TeacherName string
	DayOfWeek   int
	StartTime   string
	EndTime     string
	RoomName    string
}

func (p *Postgres) ListAcademicYears(ctx context.Context) ([]AcademicYear, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, label, semester, start_date, end_date, is_active
		FROM academic_years
		ORDER BY is_active DESC, start_date DESC, label DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []AcademicYear
	for rows.Next() {
		var year AcademicYear
		if err := rows.Scan(&year.ID, &year.Label, &year.Semester, &year.StartDate, &year.EndDate, &year.Active); err != nil {
			return nil, err
		}
		years = append(years, year)
	}
	return years, rows.Err()
}

func (p *Postgres) ActiveAcademicYear(ctx context.Context) (AcademicYear, bool) {
	var year AcademicYear
	err := p.pool.QueryRow(ctx, `
		SELECT id::text, label, semester, start_date, end_date, is_active
		FROM academic_years WHERE is_active LIMIT 1
	`).Scan(&year.ID, &year.Label, &year.Semester, &year.StartDate, &year.EndDate, &year.Active)
	if err != nil {
		return AcademicYear{}, false
	}
	return year, true
}

func (p *Postgres) CreateAcademicYear(ctx context.Context, label, semester string, start, end time.Time) (AcademicYear, error) {
	label = strings.TrimSpace(label)
	if label == "" || (semester != "Ganjil" && semester != "Genap") || end.Before(start) {
		return AcademicYear{}, errors.New("tahun ajaran tidak valid")
	}
	var year AcademicYear
	err := p.pool.QueryRow(ctx, `
		INSERT INTO academic_years (label, semester, start_date, end_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, label, semester, start_date, end_date, is_active
	`, label, semester, start, end).Scan(&year.ID, &year.Label, &year.Semester, &year.StartDate, &year.EndDate, &year.Active)
	if err != nil {
		if isUniqueViolation(err, "academic_years_label_semester_key") {
			return AcademicYear{}, ErrMasterDataExists
		}
		return AcademicYear{}, fmt.Errorf("create academic year: %w", err)
	}
	return year, nil
}

func (p *Postgres) ActivateAcademicYear(ctx context.Context, id string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE academic_years SET is_active = FALSE, updated_at = now() WHERE is_active`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE academic_years SET is_active = TRUE, updated_at = now() WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("tahun ajaran tidak ditemukan")
	}
	return tx.Commit(ctx)
}

func (p *Postgres) ListClassrooms(ctx context.Context) ([]Classroom, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT c.id::text, c.name, c.grade_level, c.major_code,
		       COALESCE(v.name, ''), COALESCE(c.homeroom_teacher_id, ''), COALESCE(u.full_name, ''),
		       COUNT(s.id)::int
		FROM classrooms c
		LEFT JOIN vocational_majors v ON v.code = c.major_code
		LEFT JOIN users u ON u.id = c.homeroom_teacher_id
		LEFT JOIN students s ON s.classroom_id = c.id AND s.is_alumni = FALSE
		GROUP BY c.id, c.name, c.grade_level, c.major_code, v.name, c.homeroom_teacher_id, u.full_name
		ORDER BY c.grade_level, c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var classes []Classroom
	for rows.Next() {
		var class Classroom
		if err := rows.Scan(&class.ID, &class.Name, &class.GradeLevel, &class.MajorCode, &class.MajorName, &class.HomeroomTeacher, &class.HomeroomName, &class.StudentCount); err != nil {
			return nil, err
		}
		classes = append(classes, class)
	}
	return classes, rows.Err()
}

func (p *Postgres) ClassroomByID(ctx context.Context, id string) (Classroom, error) {
	var class Classroom
	err := p.pool.QueryRow(ctx, `
		SELECT c.id::text, c.name, c.grade_level, c.major_code,
		       COALESCE(v.name, ''), COALESCE(c.homeroom_teacher_id, ''), COALESCE(u.full_name, ''),
		       COUNT(s.id)::int
		FROM classrooms c
		LEFT JOIN vocational_majors v ON v.code = c.major_code
		LEFT JOIN users u ON u.id = c.homeroom_teacher_id
		LEFT JOIN students s ON s.classroom_id = c.id AND s.is_alumni = FALSE
		WHERE c.id = $1::uuid
		GROUP BY c.id, c.name, c.grade_level, c.major_code, v.name, c.homeroom_teacher_id, u.full_name
	`, id).Scan(&class.ID, &class.Name, &class.GradeLevel, &class.MajorCode, &class.MajorName, &class.HomeroomTeacher, &class.HomeroomName, &class.StudentCount)
	if err != nil {
		return Classroom{}, fmt.Errorf("classroom not found: %w", err)
	}
	return class, nil
}

func (p *Postgres) CreateClassroom(ctx context.Context, name string, gradeLevel int, majorCode, homeroomTeacherID string) (Classroom, error) {
	name = strings.TrimSpace(name)
	majorCode = strings.ToUpper(strings.TrimSpace(majorCode))
	if name == "" || gradeLevel < 10 || gradeLevel > 12 || majorCode == "" {
		return Classroom{}, errors.New("kelas tidak valid")
	}
	var id string
	err := p.pool.QueryRow(ctx, `
		INSERT INTO classrooms (name, grade_level, major_code, homeroom_teacher_id)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id::text
	`, name, gradeLevel, majorCode, homeroomTeacherID).Scan(&id)
	if err != nil {
		if isUniqueViolation(err, "classrooms_name_major_key") {
			return Classroom{}, ErrMasterDataExists
		}
		return Classroom{}, fmt.Errorf("create classroom: %w", err)
	}
	return p.ClassroomByID(ctx, id)
}

func (p *Postgres) UpdateClassroom(ctx context.Context, id, name string, gradeLevel int, majorCode, homeroomTeacherID string) (Classroom, error) {
	name = strings.TrimSpace(name)
	majorCode = strings.ToUpper(strings.TrimSpace(majorCode))
	if id == "" || name == "" || gradeLevel < 10 || gradeLevel > 12 || majorCode == "" {
		return Classroom{}, errors.New("kelas tidak valid")
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Classroom{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE classrooms
		SET name = $2, grade_level = $3, major_code = $4,
		    homeroom_teacher_id = NULLIF($5, ''), updated_at = now()
		WHERE id = $1::uuid
	`, id, name, gradeLevel, majorCode, homeroomTeacherID)
	if err != nil {
		if isUniqueViolation(err, "classrooms_name_major_key") {
			return Classroom{}, ErrMasterDataExists
		}
		return Classroom{}, fmt.Errorf("update classroom: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Classroom{}, errors.New("kelas tidak ditemukan")
	}
	// Keep the denormalized display fields on student profiles in sync.
	if _, err := tx.Exec(ctx, `
		UPDATE students SET classroom_name = $2, major_code = $3, updated_at = now()
		WHERE classroom_id = $1::uuid
	`, id, name, majorCode); err != nil {
		return Classroom{}, fmt.Errorf("sync student classroom: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Classroom{}, err
	}
	return p.ClassroomByID(ctx, id)
}

func (p *Postgres) DeleteClassroom(ctx context.Context, id string) error {
	var students, schedules int
	if err := p.pool.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM students WHERE classroom_id = $1::uuid),
		       (SELECT COUNT(*) FROM schedules WHERE classroom_id = $1::uuid)
	`, id).Scan(&students, &schedules); err != nil {
		return err
	}
	if students > 0 || schedules > 0 {
		return fmt.Errorf("%w: kelas masih memiliki %d siswa dan %d jadwal", ErrMasterDataInUse, students, schedules)
	}
	tag, err := p.pool.Exec(ctx, `DELETE FROM classrooms WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete classroom: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("kelas tidak ditemukan")
	}
	return nil
}

func (p *Postgres) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT sc.id::text, sc.classroom_id::text, COALESCE(c.name, ''), sc.subject_id,
		       COALESCE(sc.teacher_id, ''), COALESCE(u.full_name, ''),
		       sc.day_of_week, to_char(sc.start_time, 'HH24:MI'), to_char(sc.end_time, 'HH24:MI'), sc.room_name
		FROM schedules sc
		LEFT JOIN classrooms c ON c.id = sc.classroom_id
		LEFT JOIN users u ON u.id = sc.teacher_id
		ORDER BY sc.day_of_week, sc.start_time, c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var schedule Schedule
		if err := rows.Scan(&schedule.ID, &schedule.ClassroomID, &schedule.ClassName, &schedule.SubjectID, &schedule.TeacherID, &schedule.TeacherName, &schedule.DayOfWeek, &schedule.StartTime, &schedule.EndTime, &schedule.RoomName); err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}
	return schedules, rows.Err()
}

func (p *Postgres) CreateSchedule(ctx context.Context, classroomID, subjectID, teacherID string, dayOfWeek int, startTime, endTime, roomName string) error {
	subjectID = strings.TrimSpace(subjectID)
	if err := validateSchedule(classroomID, subjectID, dayOfWeek, startTime, endTime); err != nil {
		return err
	}
	_, err := p.pool.Exec(ctx, `
		INSERT INTO schedules (classroom_id, subject_id, teacher_id, day_of_week, start_time, end_time, room_name)
		VALUES ($1::uuid, $2, NULLIF($3, ''), $4, $5::time, $6::time, $7)
	`, classroomID, subjectID, teacherID, dayOfWeek, startTime, endTime, strings.TrimSpace(roomName))
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

func (p *Postgres) UpdateSchedule(ctx context.Context, id, classroomID, subjectID, teacherID string, dayOfWeek int, startTime, endTime, roomName string) error {
	subjectID = strings.TrimSpace(subjectID)
	if id == "" {
		return errors.New("jadwal tidak ditemukan")
	}
	if err := validateSchedule(classroomID, subjectID, dayOfWeek, startTime, endTime); err != nil {
		return err
	}
	tag, err := p.pool.Exec(ctx, `
		UPDATE schedules
		SET classroom_id = $2::uuid, subject_id = $3, teacher_id = NULLIF($4, ''),
		    day_of_week = $5, start_time = $6::time, end_time = $7::time, room_name = $8
		WHERE id = $1::uuid
	`, id, classroomID, subjectID, teacherID, dayOfWeek, startTime, endTime, strings.TrimSpace(roomName))
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("jadwal tidak ditemukan")
	}
	return nil
}

func (p *Postgres) DeleteSchedule(ctx context.Context, id string) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("jadwal tidak ditemukan")
	}
	return nil
}

// RolloverMove promotes every active student in FromClassID to ToClassID.
type RolloverMove struct {
	FromClassID string
	ToClassID   string
}

// RolloverResult reports what one explicit academic-year rollover changed.
type RolloverResult struct {
	MovedStudents     int
	GraduatedStudents int
	TargetYear        AcademicYear
}

// RolloverStudents promotes and graduates students, then activates the target
// academic year in one transaction. Classrooms behave as class slots across
// years: grade 11 moves into the existing grade 12 slot after it graduates.
func (p *Postgres) RolloverStudents(ctx context.Context, targetYearID string, moves []RolloverMove, graduateClassIDs []string) (RolloverResult, error) {
	targetYearID = strings.TrimSpace(targetYearID)
	if targetYearID == "" || len(moves)+len(graduateClassIDs) == 0 {
		return RolloverResult{}, errors.New("pilih tahun ajaran tujuan dan minimal satu kelas")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return RolloverResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var year AcademicYear
	err = tx.QueryRow(ctx, `
		SELECT id::text, label, semester, start_date, end_date, is_active
		FROM academic_years WHERE id = $1::uuid FOR UPDATE
	`, targetYearID).Scan(&year.ID, &year.Label, &year.Semester, &year.StartDate, &year.EndDate, &year.Active)
	if err != nil {
		return RolloverResult{}, errors.New("tahun ajaran tujuan tidak ditemukan")
	}
	if year.Active {
		return RolloverResult{}, errors.New("tahun ajaran tujuan sudah aktif; pilih tahun ajaran berikutnya")
	}

	classIDs := make([]string, 0, len(moves)*2+len(graduateClassIDs))
	seenIDs := map[string]bool{}
	addID := func(id string) {
		id = strings.TrimSpace(id)
		if id != "" && !seenIDs[id] {
			seenIDs[id] = true
			classIDs = append(classIDs, id)
		}
	}
	for _, move := range moves {
		addID(move.FromClassID)
		addID(move.ToClassID)
	}
	for _, id := range graduateClassIDs {
		addID(id)
	}

	type rolloverClass struct {
		name  string
		grade int
		major string
	}
	classes := map[string]rolloverClass{}
	rows, err := tx.Query(ctx, `
		SELECT id::text, name, grade_level, major_code
		FROM classrooms WHERE id::text = ANY($1::text[]) FOR UPDATE
	`, classIDs)
	if err != nil {
		return RolloverResult{}, fmt.Errorf("lock classrooms: %w", err)
	}
	for rows.Next() {
		var id string
		var class rolloverClass
		if err := rows.Scan(&id, &class.name, &class.grade, &class.major); err != nil {
			rows.Close()
			return RolloverResult{}, err
		}
		classes[id] = class
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return RolloverResult{}, err
	}

	cleanMoves := make([]RolloverMove, 0, len(moves))
	usedSources := map[string]bool{}
	usedTargets := map[string]bool{}
	for _, move := range moves {
		from := strings.TrimSpace(move.FromClassID)
		to := strings.TrimSpace(move.ToClassID)
		source, sourceOK := classes[from]
		target, targetOK := classes[to]
		if from == "" || to == "" || !sourceOK || !targetOK {
			return RolloverResult{}, errors.New("kelas asal atau tujuan tidak ditemukan")
		}
		if from == to {
			return RolloverResult{}, errors.New("kelas asal dan tujuan tidak boleh sama")
		}
		if usedSources[from] {
			return RolloverResult{}, fmt.Errorf("kelas %s dipilih lebih dari sekali", source.name)
		}
		if usedTargets[to] {
			return RolloverResult{}, fmt.Errorf("kelas tujuan %s dipakai lebih dari sekali", target.name)
		}
		if target.grade != source.grade+1 || target.major != source.major {
			return RolloverResult{}, fmt.Errorf("kelas tujuan %s harus tingkat %d jurusan %s", target.name, source.grade+1, source.major)
		}
		usedSources[from] = true
		usedTargets[to] = true
		cleanMoves = append(cleanMoves, RolloverMove{FromClassID: from, ToClassID: to})
	}

	cleanGraduates := make([]string, 0, len(graduateClassIDs))
	for _, rawID := range graduateClassIDs {
		id := strings.TrimSpace(rawID)
		class, ok := classes[id]
		if !ok {
			return RolloverResult{}, errors.New("kelas yang diluluskan tidak ditemukan")
		}
		if usedSources[id] {
			return RolloverResult{}, fmt.Errorf("kelas %s dipilih untuk naik kelas dan lulus sekaligus", class.name)
		}
		if class.grade != 12 {
			return RolloverResult{}, fmt.Errorf("hanya kelas tingkat 12 yang bisa diluluskan (%s tingkat %d)", class.name, class.grade)
		}
		usedSources[id] = true
		cleanGraduates = append(cleanGraduates, id)
	}

	// Graduate first, then promote from the highest grade down, so a target
	// slot is vacated before the next cohort moves into it.
	result := RolloverResult{}
	for _, id := range cleanGraduates {
		tag, err := tx.Exec(ctx, `
			UPDATE students SET is_alumni = TRUE, updated_at = now()
			WHERE classroom_id = $1::uuid AND is_alumni = FALSE
		`, id)
		if err != nil {
			return RolloverResult{}, fmt.Errorf("graduate classroom: %w", err)
		}
		result.GraduatedStudents += int(tag.RowsAffected())
	}
	sort.SliceStable(cleanMoves, func(i, j int) bool {
		return classes[cleanMoves[i].FromClassID].grade > classes[cleanMoves[j].FromClassID].grade
	})
	for _, move := range cleanMoves {
		var occupied int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM students WHERE classroom_id = $1::uuid AND is_alumni = FALSE
		`, move.ToClassID).Scan(&occupied); err != nil {
			return RolloverResult{}, err
		}
		if occupied > 0 {
			target := classes[move.ToClassID]
			return RolloverResult{}, fmt.Errorf("kelas tujuan %s masih berisi %d siswa aktif; kosongkan dulu", target.name, occupied)
		}
		target := classes[move.ToClassID]
		tag, err := tx.Exec(ctx, `
			UPDATE students
			SET classroom_id = $2::uuid, classroom_name = $3, major_code = $4, updated_at = now()
			WHERE classroom_id = $1::uuid AND is_alumni = FALSE
		`, move.FromClassID, move.ToClassID, target.name, target.major)
		if err != nil {
			return RolloverResult{}, fmt.Errorf("promote students: %w", err)
		}
		result.MovedStudents += int(tag.RowsAffected())
	}
	if result.MovedStudents+result.GraduatedStudents == 0 {
		return RolloverResult{}, errors.New("tidak ada siswa aktif pada kelas yang dipilih")
	}

	if _, err := tx.Exec(ctx, `UPDATE academic_years SET is_active = FALSE, updated_at = now() WHERE is_active`); err != nil {
		return RolloverResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE academic_years SET is_active = TRUE, updated_at = now() WHERE id = $1::uuid`, targetYearID); err != nil {
		return RolloverResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RolloverResult{}, err
	}
	year.Active = true
	result.TargetYear = year
	return result, nil
}

func validateSchedule(classroomID, subjectID string, dayOfWeek int, startTime, endTime string) error {
	if classroomID == "" || subjectID == "" || dayOfWeek < 1 || dayOfWeek > 7 {
		return errors.New("jadwal tidak valid")
	}
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return errors.New("jam mulai tidak valid")
	}
	end, err := time.Parse("15:04", endTime)
	if err != nil || !end.After(start) {
		return errors.New("jam selesai harus setelah jam mulai")
	}
	return nil
}
