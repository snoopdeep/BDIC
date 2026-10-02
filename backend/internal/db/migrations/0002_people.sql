-- 0002_people.sql
-- Permanent records for students, guardians, and staff.
-- Nothing here is ever hard-deleted: a student who leaves keeps their record
-- and simply stops appearing in class lists.

CREATE TABLE staff (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid UNIQUE REFERENCES users (id) ON DELETE SET NULL,
    employee_code     text        NOT NULL,
    full_name_en      text        NOT NULL,
    full_name_hi      text,
    designation_en    text,
    designation_hi    text,
    department        text,
    qualification     text,
    date_of_joining   date,
    date_of_birth     date,
    gender            text,
    phone             text,
    email             text,
    address           text,
    emergency_contact text,
    photo_file_id     uuid REFERENCES files (id),
    show_on_website   boolean     NOT NULL DEFAULT false,
    photo_consent     boolean     NOT NULL DEFAULT false,
    is_class_teacher  boolean     NOT NULL DEFAULT false,
    status            text        NOT NULL DEFAULT 'ACTIVE',
    notes             text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT staff_employee_code_unique UNIQUE (employee_code),
    CONSTRAINT staff_status CHECK (status IN ('ACTIVE', 'ON_LEAVE', 'RESIGNED', 'RETIRED')),
    CONSTRAINT staff_gender CHECK (gender IS NULL OR gender IN ('MALE', 'FEMALE', 'OTHER'))
);

CREATE INDEX staff_status_idx ON staff (status);

-- Which subjects a teacher is allowed to enter marks for. Marks entry checks
-- this table, so a teacher cannot touch another teacher's subject.
CREATE TABLE staff_subjects (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id   uuid NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
    subject_id uuid NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    CONSTRAINT staff_subjects_unique UNIQUE (staff_id, subject_id)
);

CREATE TABLE students (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admission_no       text        NOT NULL,
    user_id            uuid UNIQUE REFERENCES users (id) ON DELETE SET NULL,
    full_name_en       text        NOT NULL,
    full_name_hi       text,
    date_of_birth      date,
    gender             text,
    category           text,
    religion           text,
    mother_tongue      text        DEFAULT 'Hindi',
    blood_group        text,
    aadhaar_last4      text,
    father_name        text,
    mother_name        text,
    father_occupation  text,
    mother_occupation  text,
    address_line       text,
    village            text,
    district           text,
    state              text        DEFAULT 'Uttar Pradesh',
    pincode            text,
    previous_school    text,
    last_class_passed  text,
    date_of_admission  date        NOT NULL DEFAULT CURRENT_DATE,
    photo_file_id      uuid REFERENCES files (id),
    photo_consent      boolean     NOT NULL DEFAULT false,
    medical_notes      text,
    status             text        NOT NULL DEFAULT 'ACTIVE',
    left_on            date,
    left_reason        text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT students_admission_no_unique UNIQUE (admission_no),
    CONSTRAINT students_status CHECK (status IN (
        'ACTIVE', 'PROMOTED', 'DETAINED', 'TRANSFERRED', 'WITHDRAWN', 'ALUMNUS'
    )),
    CONSTRAINT students_gender CHECK (gender IS NULL OR gender IN ('MALE', 'FEMALE', 'OTHER')),
    CONSTRAINT students_category CHECK (category IS NULL OR category IN ('GEN', 'OBC', 'SC', 'ST', 'EWS')),
    CONSTRAINT students_aadhaar_last4 CHECK (aadhaar_last4 IS NULL OR aadhaar_last4 ~ '^[0-9]{4}$')
);

CREATE INDEX students_status_idx ON students (status);
CREATE INDEX students_name_search ON students USING gin (to_tsvector('simple', full_name_en));

CREATE TABLE guardians (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid REFERENCES users (id) ON DELETE SET NULL,
    full_name_en  text        NOT NULL,
    full_name_hi  text,
    relation      text        NOT NULL,
    phone         text        NOT NULL,
    alt_phone     text,
    email         text,
    occupation    text,
    address       text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT guardians_relation CHECK (relation IN ('FATHER', 'MOTHER', 'GUARDIAN', 'OTHER'))
);

CREATE INDEX guardians_phone_idx ON guardians (phone);

-- One login maps to at most one guardian row, so the identity lookup that
-- joins users to guardians can never multiply rows.
CREATE UNIQUE INDEX guardians_user_unique ON guardians (user_id) WHERE user_id IS NOT NULL;

-- A parent login reaches exactly the children linked here and no others.
-- Every parent-facing query joins through this table.
CREATE TABLE student_guardians (
    student_id  uuid    NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    guardian_id uuid    NOT NULL REFERENCES guardians (id) ON DELETE CASCADE,
    is_primary  boolean NOT NULL DEFAULT false,
    PRIMARY KEY (student_id, guardian_id)
);

CREATE INDEX student_guardians_guardian ON student_guardians (guardian_id);

-- One row per student per academic session. This is what preserves history when
-- a student is promoted: last year's row stays exactly as it was.
CREATE TABLE enrollments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id   uuid NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    session_id   uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    class_id     uuid NOT NULL REFERENCES classes (id),
    section_id   uuid NOT NULL REFERENCES sections (id),
    stream_id    uuid REFERENCES streams (id),
    roll_no      int,
    status       text        NOT NULL DEFAULT 'ENROLLED',
    result       text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT enrollments_one_per_session UNIQUE (student_id, session_id),
    CONSTRAINT enrollments_roll_unique UNIQUE (session_id, section_id, roll_no),
    CONSTRAINT enrollments_status CHECK (status IN ('ENROLLED', 'COMPLETED', 'LEFT')),
    CONSTRAINT enrollments_result CHECK (result IS NULL OR result IN (
        'PASS', 'FAIL', 'SUPPLEMENTARY', 'PENDING'
    ))
);

CREATE INDEX enrollments_section ON enrollments (session_id, section_id);
CREATE INDEX enrollments_class ON enrollments (session_id, class_id);

-- Which teacher owns which section this session. Used for attendance duty,
-- report-card remarks, and parent messaging permission.
CREATE TABLE section_teachers (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid    NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    section_id uuid    NOT NULL REFERENCES sections (id) ON DELETE CASCADE,
    staff_id   uuid    NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
    is_primary boolean NOT NULL DEFAULT true,
    CONSTRAINT section_teachers_unique UNIQUE (session_id, section_id, staff_id)
);

CREATE UNIQUE INDEX section_teachers_one_primary
    ON section_teachers (session_id, section_id)
    WHERE is_primary;

CREATE TABLE student_documents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id    uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    doc_type      text        NOT NULL,
    file_id       uuid        NOT NULL REFERENCES files (id),
    status        text        NOT NULL DEFAULT 'PENDING',
    remark        text,
    verified_by   uuid REFERENCES users (id),
    verified_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT student_documents_status CHECK (status IN ('PENDING', 'VERIFIED', 'REJECTED'))
);

CREATE INDEX student_documents_student ON student_documents (student_id);

CREATE TABLE staff_documents (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    staff_id   uuid        NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
    doc_type   text        NOT NULL,
    file_id    uuid        NOT NULL REFERENCES files (id),
    remark     text,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Certificates the office issues. Recorded so the same certificate can be
-- reprinted identically, and so the school can prove what was issued.
CREATE TABLE certificates (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id     uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    kind           text        NOT NULL,
    serial_no      text        NOT NULL,
    issued_on      date        NOT NULL DEFAULT CURRENT_DATE,
    issued_by      uuid REFERENCES users (id),
    payload        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    cancelled_at   timestamptz,
    cancel_reason  text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT certificates_serial_unique UNIQUE (kind, serial_no),
    CONSTRAINT certificates_kind CHECK (kind IN (
        'BONAFIDE', 'CHARACTER', 'TRANSFER', 'ATTENDANCE', 'ID_CARD'
    ))
);

CREATE INDEX certificates_student ON certificates (student_id, issued_on DESC);
