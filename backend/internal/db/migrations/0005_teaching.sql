-- 0005_teaching.sql
-- Homework, its attachments and optional submissions, and the permanent
-- study-material shelf. Homework is dated work with a deadline; study material
-- is a reference library that stays available all year.

CREATE TABLE homework (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    section_id   uuid        NOT NULL REFERENCES sections (id) ON DELETE CASCADE,
    subject_id   uuid        NOT NULL REFERENCES subjects (id),
    staff_id     uuid        NOT NULL REFERENCES staff (id),
    title        text        NOT NULL,
    instructions text        NOT NULL,
    due_date     date        NOT NULL,
    allow_upload boolean     NOT NULL DEFAULT false,
    status       text        NOT NULL DEFAULT 'PUBLISHED',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT homework_status CHECK (status IN ('DRAFT', 'PUBLISHED', 'CLOSED'))
);

CREATE INDEX homework_section_due ON homework (section_id, due_date DESC);
CREATE INDEX homework_staff ON homework (staff_id, created_at DESC);
CREATE INDEX homework_due ON homework (due_date);

CREATE TABLE homework_attachments (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    homework_id uuid NOT NULL REFERENCES homework (id) ON DELETE CASCADE,
    file_id     uuid REFERENCES files (id),
    link_url    text,
    label       text,
    -- An attachment is either an uploaded file or an external link, not both
    -- and not neither.
    CONSTRAINT homework_attachments_one_source CHECK (
        (file_id IS NOT NULL AND link_url IS NULL)
            OR (file_id IS NULL AND link_url IS NOT NULL)
    )
);

CREATE INDEX homework_attachments_homework ON homework_attachments (homework_id);

CREATE TABLE homework_submissions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    homework_id  uuid        NOT NULL REFERENCES homework (id) ON DELETE CASCADE,
    student_id   uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    file_id      uuid REFERENCES files (id),
    note         text,
    status       text        NOT NULL DEFAULT 'SUBMITTED',
    remark       text,
    reviewed_by  uuid REFERENCES users (id),
    reviewed_at  timestamptz,
    submitted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT homework_submissions_unique UNIQUE (homework_id, student_id),
    CONSTRAINT homework_submissions_status CHECK (status IN (
        'SUBMITTED', 'SEEN', 'ACCEPTED', 'RETURNED'
    ))
);

CREATE INDEX homework_submissions_homework ON homework_submissions (homework_id);
CREATE INDEX homework_submissions_student ON homework_submissions (student_id, submitted_at DESC);

CREATE TABLE study_materials (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    class_id      uuid        NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    subject_id    uuid        NOT NULL REFERENCES subjects (id),
    title_en      text        NOT NULL,
    title_hi      text,
    description   text,
    category      text        NOT NULL DEFAULT 'NOTES',
    file_id       uuid REFERENCES files (id),
    link_url      text,
    uploaded_by   uuid REFERENCES users (id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_materials_category CHECK (category IN (
        'NOTES', 'SYLLABUS', 'WORKSHEET', 'PREVIOUS_PAPER', 'VIDEO', 'REFERENCE'
    )),
    CONSTRAINT study_materials_one_source CHECK (
        (file_id IS NOT NULL AND link_url IS NULL)
            OR (file_id IS NULL AND link_url IS NOT NULL)
    )
);

CREATE INDEX study_materials_class_subject
    ON study_materials (session_id, class_id, subject_id, created_at DESC);
