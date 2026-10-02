-- 0009_site.sql
-- The public website's editable content: pages, events, achievements, the
-- gallery, faculty listing, and downloads.
--
-- The photo-consent gate lives here. A photograph that contains identifiable
-- students has approval_status PENDING and is served by nothing until someone
-- with authority approves it.

CREATE TABLE site_pages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       text NOT NULL UNIQUE,
    title_en   text NOT NULL,
    title_hi   text NOT NULL,
    body_en    text NOT NULL DEFAULT '',
    body_hi    text NOT NULL DEFAULT '',
    is_published boolean NOT NULL DEFAULT true,
    sort_order int  NOT NULL DEFAULT 0,
    updated_by uuid REFERENCES users (id),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE gallery_albums (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug           text NOT NULL UNIQUE,
    title_en       text NOT NULL,
    title_hi       text NOT NULL,
    category       text NOT NULL DEFAULT 'CAMPUS',
    description_en text,
    description_hi text,
    event_date     date,
    is_public      boolean NOT NULL DEFAULT false,
    sort_order     int  NOT NULL DEFAULT 0,
    created_by     uuid REFERENCES users (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT gallery_albums_category CHECK (category IN (
        'CAMPUS', 'CLASSROOM', 'LABORATORY', 'LIBRARY', 'COMPUTER_ROOM',
        'SPORTS', 'FUNCTION', 'EVENT', 'OTHER'
    ))
);

CREATE TABLE photos (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    album_id          uuid        REFERENCES gallery_albums (id) ON DELETE CASCADE,
    file_id           uuid        NOT NULL REFERENCES files (id),
    caption_en        text,
    caption_hi        text,
    -- Set by whoever uploads. If true, the photograph cannot be published
    -- until a consent check has been made against every student in it.
    contains_students boolean     NOT NULL DEFAULT true,
    approval_status   text        NOT NULL DEFAULT 'PENDING',
    approved_by       uuid REFERENCES users (id),
    approved_at       timestamptz,
    reject_reason     text,
    sort_order        int         NOT NULL DEFAULT 0,
    uploaded_by       uuid REFERENCES users (id),
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT photos_approval_status CHECK (approval_status IN ('PENDING', 'APPROVED', 'REJECTED')),
    CONSTRAINT photos_reject_reason CHECK (
        approval_status <> 'REJECTED' OR reject_reason IS NOT NULL
    )
);

CREATE INDEX photos_album ON photos (album_id, sort_order);
CREATE INDEX photos_pending ON photos (approval_status, created_at DESC)
    WHERE approval_status = 'PENDING';

ALTER TABLE gallery_albums
    ADD COLUMN cover_photo_id uuid REFERENCES photos (id) ON DELETE SET NULL;

CREATE TABLE events (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     uuid REFERENCES academic_sessions (id) ON DELETE SET NULL,
    title_en       text NOT NULL,
    title_hi       text NOT NULL,
    description_en text,
    description_hi text,
    start_date     date NOT NULL,
    end_date       date,
    start_time     time,
    venue_en       text,
    venue_hi       text,
    category       text NOT NULL DEFAULT 'GENERAL',
    needs_consent  boolean NOT NULL DEFAULT false,
    is_public      boolean NOT NULL DEFAULT true,
    album_id       uuid REFERENCES gallery_albums (id) ON DELETE SET NULL,
    created_by     uuid REFERENCES users (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT events_category CHECK (category IN (
        'GENERAL', 'EXAM', 'HOLIDAY', 'SPORTS', 'CULTURAL', 'PTM', 'NATIONAL'
    )),
    CONSTRAINT events_date_order CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX events_start ON events (start_date DESC);
CREATE INDEX events_public ON events (is_public, start_date DESC) WHERE is_public;

CREATE TABLE event_participants (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     uuid    NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    student_id   uuid    NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    role         text,
    consent_given boolean NOT NULL DEFAULT false,
    consent_by   uuid REFERENCES users (id),
    consent_at   timestamptz,
    CONSTRAINT event_participants_unique UNIQUE (event_id, student_id)
);

CREATE TABLE achievements (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title_en     text NOT NULL,
    title_hi     text NOT NULL,
    student_id   uuid REFERENCES students (id) ON DELETE SET NULL,
    student_name text,
    year         int  NOT NULL,
    category     text NOT NULL DEFAULT 'ACADEMIC',
    detail_en    text,
    detail_hi    text,
    marks_or_rank text,
    photo_id     uuid REFERENCES photos (id) ON DELETE SET NULL,
    is_featured  boolean NOT NULL DEFAULT false,
    sort_order   int  NOT NULL DEFAULT 0,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT achievements_category CHECK (category IN (
        'ACADEMIC', 'SPORTS', 'CULTURAL', 'SCHOLARSHIP', 'SCHOOL'
    )),
    CONSTRAINT achievements_year CHECK (year BETWEEN 1950 AND 2100)
);

CREATE INDEX achievements_year ON achievements (year DESC, sort_order);
CREATE INDEX achievements_featured ON achievements (is_featured) WHERE is_featured;

CREATE TABLE facilities (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title_en       text NOT NULL,
    title_hi       text NOT NULL,
    description_en text,
    description_hi text,
    photo_id       uuid REFERENCES photos (id) ON DELETE SET NULL,
    icon           text,
    sort_order     int  NOT NULL DEFAULT 0,
    is_published   boolean NOT NULL DEFAULT true
);

CREATE TABLE downloads (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title_en   text NOT NULL,
    title_hi   text NOT NULL,
    category   text NOT NULL DEFAULT 'FORM',
    file_id    uuid NOT NULL REFERENCES files (id),
    sort_order int  NOT NULL DEFAULT 0,
    is_public  boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT downloads_category CHECK (category IN (
        'FORM', 'SYLLABUS', 'FEE_STRUCTURE', 'PREVIOUS_PAPER', 'POLICY', 'OTHER'
    ))
);

-- The management committee shown on the About page. Kept separate from staff
-- because committee members are usually not employees.
CREATE TABLE committee_members (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name_en   text NOT NULL,
    full_name_hi   text,
    designation_en text NOT NULL,
    designation_hi text,
    photo_id       uuid REFERENCES photos (id) ON DELETE SET NULL,
    sort_order     int  NOT NULL DEFAULT 0,
    is_published   boolean NOT NULL DEFAULT true
);
