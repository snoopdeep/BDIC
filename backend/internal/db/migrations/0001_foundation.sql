-- 0001_foundation.sql
-- School identity, academic structure, users, and the audit trail.
-- Every table that records money uses integer paise, never floating point.

CREATE TABLE schools (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name_en          text        NOT NULL,
    name_hi          text        NOT NULL,
    short_name       text        NOT NULL DEFAULT 'BDIC',
    affiliation_no   text,
    udise_code       text,
    board            text        NOT NULL DEFAULT 'UP Board (UPMSP)',
    address_en       text,
    address_hi       text,
    village          text,
    district         text,
    state            text        NOT NULL DEFAULT 'Uttar Pradesh',
    pincode          text,
    phone_primary    text,
    phone_secondary  text,
    email            text,
    office_hours_en  text,
    office_hours_hi  text,
    principal_name   text,
    logo_file_id     uuid,
    map_embed_url    text,
    instagram_url    text,
    facebook_url     text,
    google_place_url text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    -- There is exactly one school. This column exists only to carry the
    -- constraint that says so: it can hold only true, and only one row can
    -- hold it. Keeping the school in a table rather than in constants is what
    -- lets the office correct the address or a phone number without a deploy.
    singleton        boolean     NOT NULL DEFAULT true,
    CONSTRAINT schools_singleton_value CHECK (singleton),
    CONSTRAINT schools_singleton_unique UNIQUE (singleton)
);

CREATE TABLE academic_sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  uuid        NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    start_date date        NOT NULL,
    end_date   date        NOT NULL,
    is_current boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT academic_sessions_name_unique UNIQUE (school_id, name),
    CONSTRAINT academic_sessions_dates CHECK (end_date > start_date)
);

CREATE UNIQUE INDEX academic_sessions_one_current
    ON academic_sessions (school_id)
    WHERE is_current;

CREATE TABLE streams (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code       text NOT NULL UNIQUE,
    name_en    text NOT NULL,
    name_hi    text NOT NULL,
    sort_order int  NOT NULL DEFAULT 0
);

CREATE TABLE classes (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    code       text NOT NULL,
    name_en    text NOT NULL,
    name_hi    text NOT NULL,
    level      int  NOT NULL,
    has_stream boolean NOT NULL DEFAULT false,
    sort_order int  NOT NULL DEFAULT 0,
    CONSTRAINT classes_code_unique UNIQUE (school_id, code),
    CONSTRAINT classes_level_range CHECK (level BETWEEN 1 AND 12)
);

CREATE TABLE sections (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id   uuid NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    stream_id  uuid REFERENCES streams (id),
    name       text NOT NULL,
    capacity   int  NOT NULL DEFAULT 60,
    sort_order int  NOT NULL DEFAULT 0,
    CONSTRAINT sections_name_unique UNIQUE (class_id, name),
    CONSTRAINT sections_capacity_positive CHECK (capacity > 0)
);

CREATE TABLE subjects (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id   uuid NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    code        text NOT NULL,
    name_en     text NOT NULL,
    name_hi     text NOT NULL,
    is_language boolean NOT NULL DEFAULT false,
    sort_order  int  NOT NULL DEFAULT 0,
    CONSTRAINT subjects_code_unique UNIQUE (school_id, code)
);

-- Which subjects a class (optionally a stream within that class) actually runs,
-- and how many periods a week each is owed. The timetable validator reads this.
CREATE TABLE class_subjects (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    class_id       uuid NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    stream_id      uuid REFERENCES streams (id),
    subject_id     uuid NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    is_compulsory  boolean NOT NULL DEFAULT true,
    weekly_periods int     NOT NULL DEFAULT 5,
    CONSTRAINT class_subjects_unique UNIQUE (session_id, class_id, stream_id, subject_id),
    CONSTRAINT class_subjects_periods CHECK (weekly_periods BETWEEN 0 AND 40)
);

-- Uploaded files are recorded here and stored on disk (local) or S3 (deployed).
-- Nothing is served by guessing a URL: a handler checks the role first.
CREATE TABLE files (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key   text        NOT NULL UNIQUE,
    original_name text        NOT NULL,
    mime_type     text        NOT NULL,
    size_bytes    bigint      NOT NULL,
    visibility    text        NOT NULL DEFAULT 'PRIVATE',
    uploaded_by   uuid,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT files_visibility CHECK (visibility IN ('PRIVATE', 'PUBLIC')),
    CONSTRAINT files_size_positive CHECK (size_bytes > 0)
);

ALTER TABLE schools
    ADD CONSTRAINT schools_logo_file_fk FOREIGN KEY (logo_file_id) REFERENCES files (id);

CREATE TABLE users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email          text,
    phone          text,
    employee_code  text,
    admission_no   text,
    password_hash  text        NOT NULL,
    full_name_en   text        NOT NULL,
    full_name_hi   text,
    role           text        NOT NULL,
    status         text        NOT NULL DEFAULT 'ACTIVE',
    locale         text        NOT NULL DEFAULT 'hi',
    token_version  int         NOT NULL DEFAULT 1,
    must_reset     boolean     NOT NULL DEFAULT false,
    last_login_at  timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role CHECK (role IN (
        'SUPER_ADMIN', 'PRINCIPAL', 'OFFICE', 'ACCOUNTS', 'TEACHER', 'STUDENT', 'PARENT'
    )),
    CONSTRAINT users_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DISABLED')),
    CONSTRAINT users_locale CHECK (locale IN ('en', 'hi')),
    -- A user must be reachable by at least one identifier they can type at login.
    CONSTRAINT users_has_identifier CHECK (
        email IS NOT NULL OR phone IS NOT NULL
            OR employee_code IS NOT NULL OR admission_no IS NOT NULL
    )
);

CREATE UNIQUE INDEX users_email_unique ON users (lower(email)) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_phone_unique ON users (phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX users_employee_code_unique ON users (upper(employee_code)) WHERE employee_code IS NOT NULL;
CREATE UNIQUE INDEX users_admission_no_unique ON users (upper(admission_no)) WHERE admission_no IS NOT NULL;
CREATE INDEX users_role_status ON users (role, status);

-- One-time codes for password reset by OTP. Only the hash is stored, so a
-- leaked table still does not let anyone sign in.
CREATE TABLE password_reset_codes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash   text        NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    attempts    int         NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX password_reset_codes_user ON password_reset_codes (user_id, created_at DESC);

-- Who changed what, when. Never deleted, never updated. The Principal can read
-- this; nobody can edit it.
CREATE TABLE audit_log (
    id            bigserial PRIMARY KEY,
    actor_user_id uuid REFERENCES users (id),
    actor_role    text,
    action        text        NOT NULL,
    entity_type   text        NOT NULL,
    entity_id     text,
    summary       text,
    before_state  jsonb,
    after_state   jsonb,
    ip_address    inet,
    user_agent    text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_entity ON audit_log (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_log_actor ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_created ON audit_log (created_at DESC);

-- Free-form school settings the office can change without a deploy: low
-- attendance threshold, whether absence alerts are on, default language, and so on.
CREATE TABLE settings (
    key         text PRIMARY KEY,
    value       jsonb       NOT NULL,
    description text,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  uuid REFERENCES users (id)
);
