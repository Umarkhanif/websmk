-- Core schema (Blueprint V3) - minimal, extend later.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Auth & master
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash TEXT NOT NULL,
    device_fingerprint TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS vocational_majors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS classrooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    major_id UUID REFERENCES vocational_majors(id),
    grade_level INT NOT NULL,
    name TEXT NOT NULL,
    homeroom_teacher_id TEXT REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    classroom_id UUID REFERENCES classrooms(id) ON DELETE CASCADE,
    subject_id TEXT NOT NULL,
    teacher_id TEXT REFERENCES users(id),
    day_of_week INT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    room_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nisn VARCHAR(10) UNIQUE,
    nik VARCHAR(16) NOT NULL DEFAULT '',
    classroom_id UUID REFERENCES classrooms(id),
    parent_id UUID,
    is_ketua_kelas BOOLEAN NOT NULL DEFAULT FALSE,
    is_sekretaris BOOLEAN NOT NULL DEFAULT FALSE,
    is_alumni BOOLEAN NOT NULL DEFAULT FALSE,
    scholarship_type TEXT NOT NULL DEFAULT 'NONE',
    discount_percent NUMERIC NOT NULL DEFAULT 0,
    attitude_score INT NOT NULL DEFAULT 100
);

CREATE TABLE IF NOT EXISTS teachers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nip TEXT NOT NULL DEFAULT '',
    subject_ids TEXT[] NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS parent_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_wa TEXT NOT NULL DEFAULT '',
    wa_handshake_verified BOOLEAN NOT NULL DEFAULT FALSE,
    handshake_pin TEXT NOT NULL DEFAULT ''
);

-- Finance
CREATE TABLE IF NOT EXISTS tuition_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number TEXT NOT NULL UNIQUE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    period_month_year TEXT NOT NULL,
    base_amount NUMERIC NOT NULL DEFAULT 0,
    scholarship_deduction NUMERIC NOT NULL DEFAULT 0,
    final_amount NUMERIC NOT NULL DEFAULT 0,
    paid_amount NUMERIC NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'UNPAID',
    va_number TEXT NOT NULL DEFAULT '',
    gateway_provider TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES tuition_invoices(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    amount NUMERIC NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'PENDING',
    gateway_ref TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cash_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_id UUID REFERENCES payment_transactions(id) ON DELETE SET NULL,
    receipt_number TEXT NOT NULL UNIQUE,
    qr_hmac_hash TEXT NOT NULL DEFAULT '',
    pdf_r2_url TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS exam_dispensations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    exam_period TEXT NOT NULL DEFAULT '',
    clearance_type TEXT NOT NULL DEFAULT 'DISPENSATION_7D',
    reason_category TEXT NOT NULL DEFAULT '',
    agreement_doc_r2_url TEXT NOT NULL DEFAULT '',
    valid_until TIMESTAMPTZ,
    granted_by_user_id TEXT REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS reversal_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_ledger_id BIGINT,
    reason TEXT NOT NULL DEFAULT '',
    headmaster_user_id TEXT REFERENCES users(id),
    otp_hash TEXT NOT NULL DEFAULT '',
    approved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS financial_audit_ledgers (
    id BIGSERIAL PRIMARY KEY,
    invoice_id UUID REFERENCES tuition_invoices(id) ON DELETE SET NULL,
    tx_type TEXT NOT NULL,
    amount_delta NUMERIC NOT NULL DEFAULT 0,
    actor_user_id TEXT REFERENCES users(id),
    treasurer_pin_verified BOOLEAN NOT NULL DEFAULT FALSE,
    reversal_approval_id UUID REFERENCES reversal_approvals(id),
    signature_hmac TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CBT
CREATE TABLE IF NOT EXISTS exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_code CHAR(6) NOT NULL UNIQUE,
    subject_id TEXT NOT NULL,
    teacher_id TEXT REFERENCES users(id),
    duration_minutes INT NOT NULL DEFAULT 60,
    require_seb BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS exam_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    question_type TEXT NOT NULL DEFAULT 'MCQ',
    content_html_or_canvas TEXT NOT NULL DEFAULT '',
    dynamic_vars_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    options_jsonb JSONB NOT NULL DEFAULT '[]'::jsonb,
    answer_key_encrypted TEXT NOT NULL DEFAULT '',
    essay_rubric_jsonb JSONB NOT NULL DEFAULT '{}'::jsonb,
    competency_tag TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS question_difficulty_indices (
    question_id UUID NOT NULL REFERENCES exam_questions(id) ON DELETE CASCADE,
    empirical_pass_rate NUMERIC NOT NULL DEFAULT 0,
    ai_calibrated_weight NUMERIC NOT NULL DEFAULT 1,
    PRIMARY KEY (question_id)
);

CREATE TABLE IF NOT EXISTS exam_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    assigned_vars_jsonb JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ,
    server_deadline_at TIMESTAMPTZ,
    yellow_warn_count INT NOT NULL DEFAULT 0,
    lock_count INT NOT NULL DEFAULT 0,
    pg_score NUMERIC NOT NULL DEFAULT 0,
    essay_score NUMERIC NOT NULL DEFAULT 0,
    final_score NUMERIC NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS exam_violation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    violation_type TEXT NOT NULL,
    duration_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS proctor_action_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    proctor_teacher_id TEXT REFERENCES users(id),
    action_type TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exam_essay_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES exam_questions(id) ON DELETE CASCADE,
    raw_student_answer TEXT NOT NULL DEFAULT '',
    final_teacher_score NUMERIC
);

CREATE TABLE IF NOT EXISTS ai_grading_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    essay_answer_id UUID NOT NULL REFERENCES exam_essay_answers(id) ON DELETE CASCADE,
    traffic_light TEXT NOT NULL DEFAULT 'YELLOW',
    draft_score NUMERIC NOT NULL DEFAULT 0,
    ai_reasoning TEXT NOT NULL DEFAULT '',
    prompt_injection_flag BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS exam_reflections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    mastery_breakdown_jsonb JSONB NOT NULL DEFAULT '{}'::jsonb,
    three_key_takeaways_html TEXT NOT NULL DEFAULT '',
    takeaway_pdf_r2_url TEXT NOT NULL DEFAULT ''
);

-- AI & pedagogi
CREATE TABLE IF NOT EXISTS syllabi_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id TEXT NOT NULL,
    grade_level INT NOT NULL,
    pdf_r2_url TEXT NOT NULL DEFAULT '',
    extracted_text_chunks JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE IF NOT EXISTS class_journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    teach_date DATE NOT NULL DEFAULT CURRENT_DATE,
    syllabus_topic TEXT NOT NULL DEFAULT '',
    teacher_notes TEXT NOT NULL DEFAULT '',
    voice_audio_r2_url TEXT NOT NULL DEFAULT '',
    voice_transcript TEXT NOT NULL DEFAULT '',
    leader_validation_status TEXT NOT NULL DEFAULT 'PENDING',
    validated_by_student_id UUID REFERENCES students(id)
);

CREATE TABLE IF NOT EXISTS ai_lesson_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL REFERENCES class_journals(id) ON DELETE CASCADE,
    summary_html TEXT NOT NULL DEFAULT '',
    industry_case_example TEXT NOT NULL DEFAULT '',
    tokens_used INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS daily_quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL REFERENCES class_journals(id) ON DELETE CASCADE,
    question_pool_50_jsonb JSONB NOT NULL DEFAULT '[]'::jsonb,
    deadline_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS daily_quiz_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES daily_quizzes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    assigned_question_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    score NUMERIC NOT NULL DEFAULT 0,
    streak_count INT NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS student_reflections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL REFERENCES class_journals(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    understanding_level TEXT NOT NULL DEFAULT 'UNDERSTOOD',
    private_complaint_text TEXT NOT NULL DEFAULT '',
    empathy_parent_translation TEXT NOT NULL DEFAULT ''
);

-- Rapor & metrik
CREATE TABLE IF NOT EXISTS attitude_questionnaires (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    teacher_id TEXT REFERENCES users(id),
    semester_id TEXT NOT NULL DEFAULT '',
    raw_score NUMERIC NOT NULL DEFAULT 0,
    is_outlier_filtered BOOLEAN NOT NULL DEFAULT FALSE,
    effective_weight NUMERIC NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS holistic_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    subject_id TEXT NOT NULL,
    semester_id TEXT NOT NULL DEFAULT '',
    cbt_component_35 NUMERIC NOT NULL DEFAULT 0,
    quiz_component_25 NUMERIC NOT NULL DEFAULT 0,
    attitude_component_25 NUMERIC NOT NULL DEFAULT 0,
    attendance_component_15 NUMERIC NOT NULL DEFAULT 0,
    final_holistic_score NUMERIC NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS student_badges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    semester_id TEXT NOT NULL DEFAULT '',
    badge_title TEXT NOT NULL DEFAULT '',
    ai_reasoning TEXT NOT NULL DEFAULT '',
    rewind_audio_theme TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS homeroom_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    teacher_id TEXT REFERENCES users(id),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS teacher_impact_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id TEXT NOT NULL REFERENCES users(id),
    semester_id TEXT NOT NULL DEFAULT '',
    calibrated_score_growth NUMERIC NOT NULL DEFAULT 0,
    low_confusion_rate NUMERIC NOT NULL DEFAULT 0,
    leader_validation_rate NUMERIC NOT NULL DEFAULT 0,
    substitute_bonus_points NUMERIC NOT NULL DEFAULT 0,
    final_impact_rating NUMERIC NOT NULL DEFAULT 0
);

-- Kesiswaan & disiplin
CREATE TABLE IF NOT EXISTS discipline_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    points INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS student_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    article_id UUID REFERENCES discipline_articles(id),
    source_type TEXT NOT NULL DEFAULT 'MANUAL',
    points_deducted INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS parent_summons_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    letter_number TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS peer_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    article_id UUID REFERENCES discipline_articles(id),
    description TEXT NOT NULL DEFAULT '',
    evidence_r2_url TEXT NOT NULL DEFAULT '',
    distinct_cluster_count INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'PENDING_ESCROW'
);

CREATE TABLE IF NOT EXISTS peer_report_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES peer_reports(id) ON DELETE CASCADE,
    cluster_key TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS attitude_point_ledgers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    delta INT NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS counseling_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    counselor_id TEXT REFERENCES users(id),
    start_monitoring_date DATE,
    end_monitoring_date DATE,
    recovery_status TEXT NOT NULL DEFAULT 'MONITORING'
);

CREATE TABLE IF NOT EXISTS discipline_transformation_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester_id TEXT NOT NULL DEFAULT '',
    active_cases INT NOT NULL DEFAULT 0,
    recovered_count INT NOT NULL DEFAULT 0,
    relapsed_count INT NOT NULL DEFAULT 0,
    transformation_index_score NUMERIC NOT NULL DEFAULT 0
);

-- Piket, PKL, BKK, ortu, TU (minimal)
CREATE TABLE IF NOT EXISTS piket_incident_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS substitute_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id TEXT REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS pkl_placements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    company_name TEXT NOT NULL DEFAULT '',
    gps_lat NUMERIC NOT NULL DEFAULT 0,
    gps_lng NUMERIC NOT NULL DEFAULT 0,
    radius_meters INT NOT NULL DEFAULT 100,
    supervisor_pin_hash TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pkl_attendance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    placement_id UUID NOT NULL REFERENCES pkl_placements(id) ON DELETE CASCADE,
    gps_lat NUMERIC NOT NULL DEFAULT 0,
    gps_lng NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS pkl_daily_journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    placement_id UUID NOT NULL REFERENCES pkl_placements(id) ON DELETE CASCADE,
    journal_date DATE NOT NULL DEFAULT CURRENT_DATE,
    content TEXT NOT NULL DEFAULT '',
    approved BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS bkk_job_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    salary_range TEXT NOT NULL DEFAULT '',
    majors TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bkk_job_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    posting_id UUID NOT NULL REFERENCES bkk_job_postings(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'APPLIED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tracer_studies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'BEKERJA',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS education_cost_roadmaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    year INT NOT NULL,
    amount NUMERIC NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS parent_investment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    amount NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS student_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    amount NUMERIC NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS career_consultations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    topic TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS parent_leave_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    leave_type TEXT NOT NULL DEFAULT 'SAKIT',
    leave_date DATE NOT NULL DEFAULT CURRENT_DATE,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS student_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    doc_type TEXT NOT NULL DEFAULT '',
    r2_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    assignment_id TEXT NOT NULL,
    r2_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS official_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    letter_number TEXT NOT NULL DEFAULT '',
    qr_hmac_hash TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS document_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dapodik_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'OK',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS whatsapp_handshake_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_wa TEXT NOT NULL DEFAULT '',
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS executive_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    approval_type TEXT NOT NULL DEFAULT '',
    approved_by TEXT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trigger append-only ledger
CREATE OR REPLACE FUNCTION prevent_financial_ledger_tamper()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'SECURITY VIOLATION: financial_audit_ledgers is append-only.';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_lock_financial_ledger ON financial_audit_ledgers;
CREATE TRIGGER trg_lock_financial_ledger
BEFORE UPDATE OR DELETE ON financial_audit_ledgers
FOR EACH ROW EXECUTE FUNCTION prevent_financial_ledger_tamper();
