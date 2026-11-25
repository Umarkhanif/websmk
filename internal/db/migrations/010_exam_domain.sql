-- Exam cutover: app-level IDs, question metadata, and proctor session state.
ALTER TABLE exams ALTER COLUMN exam_code TYPE TEXT;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS subject_name TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS exams_app_id_key ON exams (app_id);

ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS question_number INT NOT NULL DEFAULT 0;
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS subject_name TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS kd_code TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS render_shadow_canvas BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS template_content TEXT NOT NULL DEFAULT '';
-- The legacy column is named answer_key_encrypted but has no encryption layer;
-- the cutover column keeps the current plaintext behavior explicit.
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS answer_key TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_questions ADD COLUMN IF NOT EXISTS essay_rubric TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS exam_questions_app_id_key ON exam_questions (app_id);

ALTER TABLE exam_sessions ALTER COLUMN exam_id DROP NOT NULL;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS progress TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS last_violation TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS unlock_token TEXT NOT NULL DEFAULT '';
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS emergency_2h BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS exam_sessions_app_id_key ON exam_sessions (app_id);
