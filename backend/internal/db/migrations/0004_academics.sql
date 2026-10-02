-- 0004_academics.sql
-- Periods, timetable, substitutions, holidays, and daily attendance.
-- Attendance after the day it was taken can only be changed through an
-- approved correction, and the original entry is kept.

CREATE TABLE periods (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  uuid    NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    position   int     NOT NULL,
    name_en    text    NOT NULL,
    name_hi    text    NOT NULL,
    start_time time    NOT NULL,
    end_time   time    NOT NULL,
    is_break   boolean NOT NULL DEFAULT false,
    CONSTRAINT periods_position_unique UNIQUE (school_id, position),
    CONSTRAINT periods_time_order CHECK (end_time > start_time)
);

CREATE TABLE timetable_entries (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    section_id  uuid NOT NULL REFERENCES sections (id) ON DELETE CASCADE,
    day_of_week int  NOT NULL,
    period_id   uuid NOT NULL REFERENCES periods (id) ON DELETE CASCADE,
    subject_id  uuid NOT NULL REFERENCES subjects (id),
    staff_id    uuid NOT NULL REFERENCES staff (id),
    room        text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- 1 = Monday through 6 = Saturday. Sunday is never a teaching day here.
    CONSTRAINT timetable_day_range CHECK (day_of_week BETWEEN 1 AND 6),
    -- One subject per section per period. The grid cannot hold two lessons.
    CONSTRAINT timetable_slot_unique UNIQUE (session_id, section_id, day_of_week, period_id)
);

-- The clash rule that matters: a teacher cannot be in two rooms at once.
-- Enforced by the database, not only by the screen.
CREATE UNIQUE INDEX timetable_teacher_no_clash
    ON timetable_entries (session_id, staff_id, day_of_week, period_id);

CREATE INDEX timetable_section ON timetable_entries (session_id, section_id, day_of_week);
CREATE INDEX timetable_staff ON timetable_entries (session_id, staff_id, day_of_week);

CREATE TABLE substitutions (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    date                date        NOT NULL,
    timetable_entry_id  uuid        NOT NULL REFERENCES timetable_entries (id) ON DELETE CASCADE,
    substitute_staff_id uuid        NOT NULL REFERENCES staff (id),
    reason              text,
    created_by          uuid REFERENCES users (id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT substitutions_unique UNIQUE (date, timetable_entry_id)
);

CREATE INDEX substitutions_date ON substitutions (date);
CREATE INDEX substitutions_staff ON substitutions (substitute_staff_id, date);

CREATE TABLE holidays (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    date       date NOT NULL,
    title_en   text NOT NULL,
    title_hi   text NOT NULL,
    kind       text NOT NULL DEFAULT 'HOLIDAY',
    CONSTRAINT holidays_unique UNIQUE (session_id, date),
    CONSTRAINT holidays_kind CHECK (kind IN ('HOLIDAY', 'EXAM', 'EVENT', 'VACATION'))
);

CREATE TABLE attendance (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    student_id   uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    section_id   uuid        NOT NULL REFERENCES sections (id),
    date         date        NOT NULL,
    status       text        NOT NULL,
    marked_by    uuid REFERENCES users (id),
    marked_at    timestamptz NOT NULL DEFAULT now(),
    note         text,
    CONSTRAINT attendance_one_per_day UNIQUE (student_id, date),
    CONSTRAINT attendance_status CHECK (status IN (
        'PRESENT', 'ABSENT', 'LATE', 'LEAVE', 'MEDICAL', 'HALF_DAY'
    ))
);

CREATE INDEX attendance_section_date ON attendance (section_id, date);
CREATE INDEX attendance_student_date ON attendance (student_id, date DESC);
CREATE INDEX attendance_date ON attendance (date);

-- A correction to a past day is a request, not an edit. The old value survives
-- in old_status so the register cannot be quietly rewritten.
CREATE TABLE attendance_corrections (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    attendance_id uuid        NOT NULL REFERENCES attendance (id) ON DELETE CASCADE,
    old_status    text        NOT NULL,
    new_status    text        NOT NULL,
    reason        text        NOT NULL,
    requested_by  uuid        NOT NULL REFERENCES users (id),
    requested_at  timestamptz NOT NULL DEFAULT now(),
    decided_by    uuid REFERENCES users (id),
    decided_at    timestamptz,
    decision_note text,
    status        text        NOT NULL DEFAULT 'PENDING',
    CONSTRAINT attendance_corrections_status CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'))
);

CREATE INDEX attendance_corrections_pending
    ON attendance_corrections (status, requested_at DESC);

-- Recorded when a teacher presses Save, so management can see which classes
-- have not been marked today without scanning every student row.
CREATE TABLE attendance_submissions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    section_id  uuid        NOT NULL REFERENCES sections (id) ON DELETE CASCADE,
    date        date        NOT NULL,
    marked_by   uuid REFERENCES users (id),
    present     int         NOT NULL DEFAULT 0,
    absent      int         NOT NULL DEFAULT 0,
    other       int         NOT NULL DEFAULT 0,
    submitted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT attendance_submissions_unique UNIQUE (section_id, date)
);

CREATE INDEX attendance_submissions_date ON attendance_submissions (date DESC);
