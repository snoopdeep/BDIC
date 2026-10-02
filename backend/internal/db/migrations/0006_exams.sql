-- 0006_exams.sql
-- Examinations, marks, and the approval gate.
-- A result is invisible to students and parents until the Principal publishes
-- the exam. That gate is the exam's own status column, checked on every read.

CREATE TABLE grading_scales (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    grade       text NOT NULL,
    min_percent numeric(5, 2) NOT NULL,
    max_percent numeric(5, 2) NOT NULL,
    remark_en   text,
    remark_hi   text,
    sort_order  int  NOT NULL DEFAULT 0,
    CONSTRAINT grading_scales_unique UNIQUE (session_id, grade),
    CONSTRAINT grading_scales_range CHECK (max_percent >= min_percent),
    CONSTRAINT grading_scales_bounds CHECK (
        min_percent >= 0 AND max_percent <= 100
    )
);

CREATE TABLE exams (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    name_en       text        NOT NULL,
    name_hi       text        NOT NULL,
    term          text,
    -- How much this exam counts towards the final result for the year.
    weight_percent numeric(5, 2) NOT NULL DEFAULT 100,
    status        text        NOT NULL DEFAULT 'DRAFT',
    approved_by   uuid REFERENCES users (id),
    approved_at   timestamptz,
    published_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT exams_name_unique UNIQUE (session_id, name_en),
    CONSTRAINT exams_status CHECK (status IN (
        'DRAFT', 'SCHEDULED', 'MARKS_OPEN', 'IN_REVIEW', 'APPROVED', 'PUBLISHED'
    )),
    CONSTRAINT exams_weight CHECK (weight_percent > 0 AND weight_percent <= 100)
);

CREATE INDEX exams_session_status ON exams (session_id, status);

CREATE TABLE exam_subjects (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id    uuid NOT NULL REFERENCES exams (id) ON DELETE CASCADE,
    class_id   uuid NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    stream_id  uuid REFERENCES streams (id),
    subject_id uuid NOT NULL REFERENCES subjects (id),
    exam_date  date,
    start_time time,
    max_marks  int  NOT NULL,
    pass_marks int  NOT NULL,
    -- Set when the subject teacher submits. Marks then need a review to reopen.
    locked_at  timestamptz,
    locked_by  uuid REFERENCES users (id),
    CONSTRAINT exam_subjects_unique UNIQUE (exam_id, class_id, stream_id, subject_id),
    CONSTRAINT exam_subjects_marks CHECK (max_marks > 0 AND pass_marks >= 0 AND pass_marks <= max_marks)
);

CREATE INDEX exam_subjects_exam ON exam_subjects (exam_id, class_id);

CREATE TABLE marks (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_subject_id uuid        NOT NULL REFERENCES exam_subjects (id) ON DELETE CASCADE,
    student_id      uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    -- NULL whenever special_status is not NONE. An absent student is not a zero.
    marks_obtained  numeric(6, 2),
    special_status  text        NOT NULL DEFAULT 'NONE',
    entered_by      uuid REFERENCES users (id),
    entered_at      timestamptz NOT NULL DEFAULT now(),
    reviewed_by     uuid REFERENCES users (id),
    reviewed_at     timestamptz,
    remark          text,
    CONSTRAINT marks_unique UNIQUE (exam_subject_id, student_id),
    CONSTRAINT marks_special_status CHECK (special_status IN (
        'NONE', 'ABSENT', 'EXEMPT', 'MEDICAL', 'REEXAM_PENDING'
    )),
    CONSTRAINT marks_value_matches_status CHECK (
        (special_status = 'NONE' AND marks_obtained IS NOT NULL)
            OR (special_status <> 'NONE' AND marks_obtained IS NULL)
    ),
    CONSTRAINT marks_not_negative CHECK (marks_obtained IS NULL OR marks_obtained >= 0)
);

CREATE INDEX marks_student ON marks (student_id);
CREATE INDEX marks_exam_subject ON marks (exam_subject_id);

-- When a reviewer sends a subject back to the teacher, the reason is recorded
-- rather than passed on by word of mouth.
CREATE TABLE marks_review_notes (
    id              bigserial PRIMARY KEY,
    exam_subject_id uuid        NOT NULL REFERENCES exam_subjects (id) ON DELETE CASCADE,
    action          text        NOT NULL,
    note            text,
    actor_user_id   uuid REFERENCES users (id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT marks_review_notes_action CHECK (action IN ('SUBMITTED', 'RETURNED', 'ACCEPTED'))
);

CREATE INDEX marks_review_notes_subject ON marks_review_notes (exam_subject_id, created_at DESC);

-- A generated report card is recorded so a reprint is byte-identical to what
-- the parent already received, even if a mark is corrected later.
CREATE TABLE report_cards (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id     uuid        NOT NULL REFERENCES exams (id) ON DELETE CASCADE,
    student_id  uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    snapshot    jsonb       NOT NULL,
    total_marks numeric(8, 2),
    max_marks   numeric(8, 2),
    percentage  numeric(5, 2),
    grade       text,
    division    text,
    class_rank  int,
    remark      text,
    generated_by uuid REFERENCES users (id),
    generated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT report_cards_unique UNIQUE (exam_id, student_id)
);

CREATE INDEX report_cards_student ON report_cards (student_id, generated_at DESC);
