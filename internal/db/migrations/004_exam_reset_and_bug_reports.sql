-- Migration 004: Exam reset audit logs and bug reports system
CREATE TABLE IF NOT EXISTS bug_reports (
    id VARCHAR(64) PRIMARY KEY,
    reporter_user_id VARCHAR(64) NOT NULL,
    reporter_name VARCHAR(128) NOT NULL,
    reporter_role VARCHAR(32) NOT NULL,
    category VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    page_url VARCHAR(255) NOT NULL,
    severity VARCHAR(32) NOT NULL DEFAULT 'medium',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    developer_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bug_reports_status ON bug_reports(status);
CREATE INDEX IF NOT EXISTS idx_bug_reports_created ON bug_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bug_reports_reporter ON bug_reports(reporter_user_id);

CREATE TABLE IF NOT EXISTS exam_reset_logs (
    id VARCHAR(64) PRIMARY KEY,
    student_id VARCHAR(64) NOT NULL,
    student_name VARCHAR(128) NOT NULL,
    subject_id VARCHAR(64) NOT NULL,
    proctor_user_id VARCHAR(64) NOT NULL,
    proctor_name VARCHAR(128) NOT NULL,
    reason TEXT NOT NULL,
    is_bulk BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_reset_logs_student ON exam_reset_logs(student_id);
CREATE INDEX IF NOT EXISTS idx_exam_reset_logs_created ON exam_reset_logs(created_at DESC);
