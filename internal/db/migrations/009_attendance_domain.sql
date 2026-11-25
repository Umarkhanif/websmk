-- Daily attendance cutover from app_state.attendances.
CREATE TABLE IF NOT EXISTS attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id TEXT NOT NULL UNIQUE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    schedule_id TEXT NOT NULL DEFAULT '',
    student_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    verified_source TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS attendances_student_idx ON attendances (student_id);
CREATE INDEX IF NOT EXISTS attendances_updated_idx ON attendances (updated_at DESC);
