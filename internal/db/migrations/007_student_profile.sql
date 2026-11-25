-- Student profile cutover: keep the app-level student ID and the display
-- fields that used to live only in app_state.students.
ALTER TABLE students ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE students ADD COLUMN IF NOT EXISTS full_name TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS classroom_name TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS major_code TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS parent_name TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS parent_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS bmw_choice TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS homeroom_note TEXT NOT NULL DEFAULT '';
ALTER TABLE students ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- app_id is the ID referenced by users.student_profile_id. Old rows are filled
-- by the boot backfill; the partial index tolerates NULL during that window.
CREATE UNIQUE INDEX IF NOT EXISTS students_app_id_key
    ON students (app_id) WHERE app_id IS NOT NULL AND app_id <> '';
CREATE INDEX IF NOT EXISTS students_user_id_idx ON students (user_id);
