-- 0011_academic_records.sql
-- Permanent student academic records. This was added after the original
-- academics migration had already been applied, so it must be its own
-- immutable migration rather than changing 0004 in place.

CREATE TABLE academic_records (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      uuid          NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    subject         text          NOT NULL,
    academic_year   text          NOT NULL,
    session         text          NOT NULL,
    marks_obtained  numeric(6, 2) NOT NULL,
    max_marks       numeric(6, 2) NOT NULL,
    grade           text          NOT NULL,
    record_type     text          NOT NULL DEFAULT 'EXAM',
    created_by      uuid REFERENCES users (id),
    created_at      timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT academic_records_record_type CHECK (record_type IN ('EXAM', 'TRANSCRIPT')),
    CONSTRAINT academic_records_marks_valid CHECK (
        marks_obtained >= 0 AND max_marks > 0 AND marks_obtained <= max_marks
    )
);

CREATE INDEX academic_records_student
    ON academic_records (student_id, academic_year DESC);
CREATE INDEX academic_records_year_session
    ON academic_records (academic_year, session);
