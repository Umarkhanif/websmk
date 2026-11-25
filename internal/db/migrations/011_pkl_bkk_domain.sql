-- PKL & BKK cutover: app-level IDs and the fields the portals display.
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS industry_sector TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS supervisor_name TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS supervisor_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_placements ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE UNIQUE INDEX IF NOT EXISTS pkl_placements_app_id_key ON pkl_placements (app_id);

ALTER TABLE pkl_attendance_logs ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE pkl_attendance_logs ADD COLUMN IF NOT EXISTS distance_meters INT NOT NULL DEFAULT 0;
ALTER TABLE pkl_attendance_logs ADD COLUMN IF NOT EXISTS is_within_radius BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pkl_attendance_logs ADD COLUMN IF NOT EXISTS selfie_url TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_attendance_logs ADD COLUMN IF NOT EXISTS checkin_time TIMESTAMPTZ;
CREATE UNIQUE INDEX IF NOT EXISTS pkl_attendance_logs_app_id_key ON pkl_attendance_logs (app_id);

ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS activity_title TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS competency_learned TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS work_photo_url TEXT NOT NULL DEFAULT '';
ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS supervisor_approved BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pkl_daily_journals ADD COLUMN IF NOT EXISTS teacher_approved BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS pkl_daily_journals_app_id_key ON pkl_daily_journals (app_id);

ALTER TABLE bkk_job_postings ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE bkk_job_postings ADD COLUMN IF NOT EXISTS deadline_date DATE;
ALTER TABLE bkk_job_postings ADD COLUMN IF NOT EXISTS requirements TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE bkk_job_postings ADD COLUMN IF NOT EXISTS applicant_count INT NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX IF NOT EXISTS bkk_job_postings_app_id_key ON bkk_job_postings (app_id);

ALTER TABLE bkk_job_applications ADD COLUMN IF NOT EXISTS app_id TEXT;
ALTER TABLE bkk_job_applications ADD COLUMN IF NOT EXISTS applied_date DATE;
CREATE UNIQUE INDEX IF NOT EXISTS bkk_job_applications_app_id_key ON bkk_job_applications (app_id);
