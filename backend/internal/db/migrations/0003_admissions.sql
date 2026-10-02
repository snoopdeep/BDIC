-- 0003_admissions.sql
-- The admission pipeline, from a parent's phone to an enrolled student.
-- An application is never edited in place once decided: the decision, its
-- reason, and who made it are all recorded.

CREATE TABLE admission_applications (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_no     text        NOT NULL,
    session_id         uuid        NOT NULL REFERENCES academic_sessions (id),
    applicant_name     text        NOT NULL,
    date_of_birth      date,
    gender             text,
    category           text,
    religion           text,
    class_applied_id   uuid        NOT NULL REFERENCES classes (id),
    stream_id          uuid REFERENCES streams (id),
    father_name        text        NOT NULL,
    mother_name        text,
    guardian_name      text,
    guardian_relation  text,
    guardian_phone     text        NOT NULL,
    guardian_alt_phone text,
    guardian_email     text,
    address_line       text,
    village            text,
    district           text,
    state              text        DEFAULT 'Uttar Pradesh',
    pincode            text,
    previous_school    text,
    last_class_passed  text,
    last_class_percent numeric(5, 2),
    photo_file_id      uuid REFERENCES files (id),
    -- The parent proves who they are with the application number plus this phone.
    -- Not a password: the office can read it out over the counter if needed.
    lookup_token       text        NOT NULL,
    status             text        NOT NULL DEFAULT 'SUBMITTED',
    status_reason      text,
    test_date          date,
    test_time          text,
    test_venue         text,
    decided_by         uuid REFERENCES users (id),
    decided_at         timestamptz,
    converted_student_id uuid REFERENCES students (id),
    submitted_at       timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT admission_applications_no_unique UNIQUE (application_no),
    CONSTRAINT admission_applications_status CHECK (status IN (
        'SUBMITTED', 'UNDER_REVIEW', 'DOCUMENTS_NEEDED', 'TEST_SCHEDULED',
        'SELECTED', 'WAITLISTED', 'REJECTED', 'FEE_PENDING', 'ADMITTED', 'WITHDRAWN'
    )),
    CONSTRAINT admission_applications_gender CHECK (gender IS NULL OR gender IN ('MALE', 'FEMALE', 'OTHER')),
    CONSTRAINT admission_applications_category CHECK (category IS NULL OR category IN ('GEN', 'OBC', 'SC', 'ST', 'EWS')),
    CONSTRAINT admission_applications_phone CHECK (guardian_phone ~ '^[0-9]{10}$')
);

CREATE INDEX admission_applications_status ON admission_applications (status, submitted_at DESC);
CREATE INDEX admission_applications_session ON admission_applications (session_id, class_applied_id);
CREATE INDEX admission_applications_phone ON admission_applications (guardian_phone);

CREATE TABLE admission_documents (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid        NOT NULL REFERENCES admission_applications (id) ON DELETE CASCADE,
    doc_type       text        NOT NULL,
    file_id        uuid        NOT NULL REFERENCES files (id),
    status         text        NOT NULL DEFAULT 'PENDING',
    remark         text,
    reviewed_by    uuid REFERENCES users (id),
    reviewed_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT admission_documents_status CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED'))
);

CREATE INDEX admission_documents_application ON admission_documents (application_id);

-- Every status change, so a parent asking "what happened to my application"
-- gets a real answer and the office can see who moved it.
CREATE TABLE admission_events (
    id             bigserial PRIMARY KEY,
    application_id uuid        NOT NULL REFERENCES admission_applications (id) ON DELETE CASCADE,
    from_status    text,
    to_status      text        NOT NULL,
    note           text,
    actor_user_id  uuid REFERENCES users (id),
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admission_events_application ON admission_events (application_id, created_at);

-- Which documents the school requires, per class. The office edits this list;
-- the public form reads it. No code change to add a required document.
CREATE TABLE admission_required_documents (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id   uuid REFERENCES classes (id) ON DELETE CASCADE,
    doc_type   text    NOT NULL,
    label_en   text    NOT NULL,
    label_hi   text    NOT NULL,
    mandatory  boolean NOT NULL DEFAULT true,
    sort_order int     NOT NULL DEFAULT 0,
    CONSTRAINT admission_required_documents_unique UNIQUE (class_id, doc_type)
);

-- Public admission enquiries from the contact form. Lighter than an application:
-- a name, a phone, and a question.
CREATE TABLE enquiries (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name               text        NOT NULL,
    phone              text        NOT NULL,
    email              text,
    class_of_interest  uuid REFERENCES classes (id),
    message            text,
    source             text        NOT NULL DEFAULT 'WEBSITE',
    status             text        NOT NULL DEFAULT 'NEW',
    handled_by         uuid REFERENCES users (id),
    handled_at         timestamptz,
    handler_note       text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT enquiries_status CHECK (status IN ('NEW', 'CONTACTED', 'CONVERTED', 'CLOSED')),
    CONSTRAINT enquiries_phone CHECK (phone ~ '^[0-9]{10}$')
);

CREATE INDEX enquiries_status ON enquiries (status, created_at DESC);

-- Admission numbers are issued from a per-session counter so the format the
-- school approves (for example BDIC/2026/0147) stays gapless and predictable.
CREATE TABLE number_series (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       text NOT NULL,
    period      text NOT NULL,
    prefix      text NOT NULL DEFAULT '',
    next_value  int  NOT NULL DEFAULT 1,
    pad_width   int  NOT NULL DEFAULT 4,
    CONSTRAINT number_series_unique UNIQUE (scope, period),
    CONSTRAINT number_series_scope CHECK (scope IN (
        'ADMISSION_NO', 'APPLICATION_NO', 'RECEIPT_NO', 'INVOICE_NO',
        'CERT_BONAFIDE', 'CERT_CHARACTER', 'CERT_TRANSFER', 'CERT_ATTENDANCE'
    )),
    CONSTRAINT number_series_pad CHECK (pad_width BETWEEN 1 AND 10)
);
