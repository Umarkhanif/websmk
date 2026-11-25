-- School master data that used to live only in app_state.blob.
-- Academic years are explicit rows so activation and rollover are auditable.
CREATE TABLE IF NOT EXISTS academic_years (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label TEXT NOT NULL,
    semester TEXT NOT NULL CHECK (semester IN ('Ganjil', 'Genap')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (label, semester)
);

-- At most one active academic year/semester.
CREATE UNIQUE INDEX IF NOT EXISTS academic_years_one_active_idx
    ON academic_years (is_active) WHERE is_active;

INSERT INTO academic_years (label, semester, start_date, end_date, is_active)
SELECT '2026/2027', 'Ganjil', DATE '2026-07-01', DATE '2026-12-31', TRUE
WHERE NOT EXISTS (SELECT 1 FROM academic_years);

ALTER TABLE classrooms ADD COLUMN IF NOT EXISTS major_code TEXT NOT NULL DEFAULT '';
ALTER TABLE classrooms ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Classroom names are unique within a major. Existing rows get a blank major
-- code, so the index is safe to add before the operator fills master data.
CREATE UNIQUE INDEX IF NOT EXISTS classrooms_name_major_key
    ON classrooms (name, major_code);
CREATE INDEX IF NOT EXISTS classrooms_major_code_idx ON classrooms (major_code);
