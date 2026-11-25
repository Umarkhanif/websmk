-- Portal operations that were still backed only by demo data.

ALTER TABLE attendances ADD COLUMN IF NOT EXISTS attendance_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE attendances ADD COLUMN IF NOT EXISTS classroom_id UUID REFERENCES classrooms(id) ON DELETE SET NULL;

UPDATE attendances
SET attendance_date = (updated_at AT TIME ZONE 'Asia/Jakarta')::date
WHERE attendance_date = CURRENT_DATE AND updated_at::date <> CURRENT_DATE;

UPDATE attendances a
SET classroom_id = s.classroom_id
FROM students s
WHERE s.id = a.student_id AND a.classroom_id IS NULL;

CREATE INDEX IF NOT EXISTS attendances_class_date_idx
    ON attendances (classroom_id, attendance_date, updated_at DESC);

ALTER TABLE class_journals ADD COLUMN IF NOT EXISTS classroom_id UUID REFERENCES classrooms(id) ON DELETE SET NULL;
ALTER TABLE class_journals ADD COLUMN IF NOT EXISTS teacher_id TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE class_journals ADD COLUMN IF NOT EXISTS subject_id TEXT NOT NULL DEFAULT '';
ALTER TABLE class_journals ADD COLUMN IF NOT EXISTS ai_smart_notes TEXT NOT NULL DEFAULT '';
ALTER TABLE class_journals ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS class_journals_schedule_date_key
    ON class_journals (schedule_id, teach_date) WHERE schedule_id IS NOT NULL;

ALTER TABLE substitute_assignments ADD COLUMN IF NOT EXISTS schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL;
ALTER TABLE substitute_assignments ADD COLUMN IF NOT EXISTS assignment_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE substitute_assignments ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';
ALTER TABLE substitute_assignments ADD COLUMN IF NOT EXISTS assigned_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE substitute_assignments ADD COLUMN IF NOT EXISTS original_teacher_id TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS substitute_assignments_date_idx
    ON substitute_assignments (assignment_date DESC, assigned_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS substitute_assignments_slot_date_key
    ON substitute_assignments (schedule_id, assignment_date) WHERE schedule_id IS NOT NULL;

ALTER TABLE parent_leave_requests ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'DIAJUKAN';
CREATE INDEX IF NOT EXISTS parent_leave_requests_student_date_idx
    ON parent_leave_requests (student_id, leave_date DESC);

CREATE TABLE IF NOT EXISTS gate_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    nisn TEXT NOT NULL DEFAULT '',
    student_name TEXT NOT NULL DEFAULT '',
    log_type TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    pass_code TEXT NOT NULL DEFAULT '',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS gate_logs_recorded_idx ON gate_logs (recorded_at DESC);
CREATE INDEX IF NOT EXISTS gate_logs_nisn_idx ON gate_logs (nisn);
