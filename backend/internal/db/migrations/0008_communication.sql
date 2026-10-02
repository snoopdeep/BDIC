-- 0008_communication.sql
-- Notices, acknowledgements, parent-teacher messaging, and the outbound queue.
--
-- Nothing in this file sends anything. outbound_messages holds DRAFT rows that
-- show the school exactly what would be sent. A provider is connected later by
-- filling the credentials in .env and switching the dispatcher on.

CREATE TABLE notices (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title_en      text        NOT NULL,
    title_hi      text,
    body_en       text        NOT NULL,
    body_hi       text,
    audience_kind text        NOT NULL,
    -- Section ids, class ids, or user ids, depending on audience_kind.
    audience_refs uuid[]      NOT NULL DEFAULT '{}',
    audience_roles text[]     NOT NULL DEFAULT '{}',
    requires_ack  boolean     NOT NULL DEFAULT false,
    is_public     boolean     NOT NULL DEFAULT false,
    publish_at    timestamptz NOT NULL DEFAULT now(),
    expire_at     timestamptz,
    status        text        NOT NULL DEFAULT 'DRAFT',
    created_by    uuid REFERENCES users (id),
    approved_by   uuid REFERENCES users (id),
    approved_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notices_audience_kind CHECK (audience_kind IN (
        'WHOLE_SCHOOL', 'CLASSES', 'SECTIONS', 'ROLES', 'INDIVIDUALS'
    )),
    CONSTRAINT notices_status CHECK (status IN (
        'DRAFT', 'PENDING_APPROVAL', 'SCHEDULED', 'PUBLISHED', 'EXPIRED', 'WITHDRAWN'
    )),
    CONSTRAINT notices_expiry_after_publish CHECK (
        expire_at IS NULL OR expire_at > publish_at
    )
);

CREATE INDEX notices_published ON notices (status, publish_at DESC);
CREATE INDEX notices_public ON notices (is_public, publish_at DESC) WHERE is_public;

CREATE TABLE notice_attachments (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id uuid NOT NULL REFERENCES notices (id) ON DELETE CASCADE,
    file_id   uuid NOT NULL REFERENCES files (id),
    label     text
);

CREATE INDEX notice_attachments_notice ON notice_attachments (notice_id);

-- Who has read it and who has acknowledged it, so the sender can chase only
-- the people who have not.
CREATE TABLE notice_receipts (
    notice_id       uuid        NOT NULL REFERENCES notices (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at         timestamptz,
    acknowledged_at timestamptz,
    PRIMARY KEY (notice_id, user_id)
);

CREATE INDEX notice_receipts_user ON notice_receipts (user_id, read_at);

-- A thread is always tied to one student, which is what gives a teacher the
-- right to be in it and a guardian the right to read it.
CREATE TABLE message_threads (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    staff_id    uuid        NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
    guardian_id uuid        NOT NULL REFERENCES guardians (id) ON DELETE CASCADE,
    subject     text,
    status      text        NOT NULL DEFAULT 'OPEN',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_threads_unique UNIQUE (student_id, staff_id, guardian_id),
    CONSTRAINT message_threads_status CHECK (status IN ('OPEN', 'CLOSED'))
);

CREATE INDEX message_threads_staff ON message_threads (staff_id, updated_at DESC);
CREATE INDEX message_threads_guardian ON message_threads (guardian_id, updated_at DESC);

CREATE TABLE messages (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id      uuid        NOT NULL REFERENCES message_threads (id) ON DELETE CASCADE,
    sender_user_id uuid        NOT NULL REFERENCES users (id),
    body           text        NOT NULL,
    read_at        timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX messages_thread ON messages (thread_id, created_at);

-- The outbound queue. Every row is what WOULD be sent. Nothing leaves the
-- building while status is DRAFT, which is the default and the only status the
-- local build ever writes.
CREATE TABLE outbound_messages (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    channel       text        NOT NULL,
    recipient     text        NOT NULL,
    recipient_user_id uuid REFERENCES users (id),
    template_key  text,
    subject       text,
    body          text        NOT NULL,
    payload       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    status        text        NOT NULL DEFAULT 'DRAFT',
    provider      text,
    provider_ref  text,
    error         text,
    attempts      int         NOT NULL DEFAULT 0,
    related_type  text,
    related_id    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    sent_at       timestamptz,
    CONSTRAINT outbound_messages_channel CHECK (channel IN ('SMS', 'WHATSAPP', 'EMAIL', 'PUSH')),
    CONSTRAINT outbound_messages_status CHECK (status IN (
        'DRAFT', 'QUEUED', 'SENT', 'FAILED', 'CANCELLED'
    ))
);

CREATE INDEX outbound_messages_status ON outbound_messages (status, created_at DESC);
CREATE INDEX outbound_messages_channel ON outbound_messages (channel, created_at DESC);
CREATE INDEX outbound_messages_related ON outbound_messages (related_type, related_id);
