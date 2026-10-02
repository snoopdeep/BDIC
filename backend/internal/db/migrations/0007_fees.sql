-- 0007_fees.sql
-- Fee structures, invoices, receipts, and concessions.
--
-- Every amount is an integer number of PAISE. There is no floating point
-- anywhere in this file, because 0.1 + 0.2 is not 0.3 and a fee ledger must
-- balance to the rupee.
--
-- A receipt is never edited or deleted. A wrong entry is cancelled with a
-- reason, and a fresh receipt is issued, so the number series stays gapless
-- and the trail stays intact.

CREATE TABLE fee_heads (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id    uuid    NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    code         text    NOT NULL,
    name_en      text    NOT NULL,
    name_hi      text    NOT NULL,
    is_recurring boolean NOT NULL DEFAULT true,
    sort_order   int     NOT NULL DEFAULT 0,
    CONSTRAINT fee_heads_code_unique UNIQUE (school_id, code)
);

CREATE TABLE fee_plans (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    class_id   uuid NOT NULL REFERENCES classes (id) ON DELETE CASCADE,
    stream_id  uuid REFERENCES streams (id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fee_plans_unique UNIQUE (session_id, class_id, stream_id)
);

CREATE TABLE fee_plan_items (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    fee_plan_id    uuid NOT NULL REFERENCES fee_plans (id) ON DELETE CASCADE,
    fee_head_id    uuid NOT NULL REFERENCES fee_heads (id),
    installment_no int  NOT NULL DEFAULT 1,
    amount_paise   bigint NOT NULL,
    due_date       date NOT NULL,
    CONSTRAINT fee_plan_items_unique UNIQUE (fee_plan_id, fee_head_id, installment_no),
    CONSTRAINT fee_plan_items_amount CHECK (amount_paise > 0),
    CONSTRAINT fee_plan_items_installment CHECK (installment_no BETWEEN 1 AND 12)
);

CREATE INDEX fee_plan_items_plan ON fee_plan_items (fee_plan_id, installment_no);

CREATE TABLE late_fee_rules (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        uuid   NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    grace_days        int    NOT NULL DEFAULT 0,
    amount_paise      bigint NOT NULL DEFAULT 0,
    per_day           boolean NOT NULL DEFAULT false,
    max_amount_paise  bigint,
    CONSTRAINT late_fee_rules_one_per_session UNIQUE (session_id),
    CONSTRAINT late_fee_rules_amounts CHECK (
        amount_paise >= 0 AND grace_days >= 0
            AND (max_amount_paise IS NULL OR max_amount_paise >= amount_paise)
    )
);

CREATE TABLE concession_types (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id         uuid    NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    code              text    NOT NULL,
    name_en           text    NOT NULL,
    name_hi           text    NOT NULL,
    kind              text    NOT NULL,
    default_value     numeric(10, 2) NOT NULL DEFAULT 0,
    requires_approval boolean NOT NULL DEFAULT true,
    CONSTRAINT concession_types_code_unique UNIQUE (school_id, code),
    CONSTRAINT concession_types_kind CHECK (kind IN ('PERCENT', 'AMOUNT')),
    CONSTRAINT concession_types_value CHECK (default_value >= 0)
);

CREATE TABLE student_concessions (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id         uuid    NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    session_id         uuid    NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    concession_type_id uuid    NOT NULL REFERENCES concession_types (id),
    fee_head_id        uuid REFERENCES fee_heads (id),
    kind               text    NOT NULL,
    value              numeric(10, 2) NOT NULL,
    reason             text    NOT NULL,
    status             text    NOT NULL DEFAULT 'PENDING',
    requested_by       uuid REFERENCES users (id),
    approved_by        uuid REFERENCES users (id),
    approved_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT student_concessions_kind CHECK (kind IN ('PERCENT', 'AMOUNT')),
    CONSTRAINT student_concessions_status CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'WITHDRAWN')),
    CONSTRAINT student_concessions_value CHECK (value >= 0)
);

CREATE INDEX student_concessions_student ON student_concessions (student_id, session_id);
CREATE INDEX student_concessions_pending ON student_concessions (status) WHERE status = 'PENDING';

CREATE TABLE invoices (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_no       text        NOT NULL,
    student_id       uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    session_id       uuid        NOT NULL REFERENCES academic_sessions (id) ON DELETE CASCADE,
    installment_no   int         NOT NULL DEFAULT 1,
    due_date         date        NOT NULL,
    gross_paise      bigint      NOT NULL DEFAULT 0,
    concession_paise bigint      NOT NULL DEFAULT 0,
    late_fee_paise   bigint      NOT NULL DEFAULT 0,
    paid_paise       bigint      NOT NULL DEFAULT 0,
    status           text        NOT NULL DEFAULT 'DUE',
    cancelled_at     timestamptz,
    cancel_reason    text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT invoices_no_unique UNIQUE (invoice_no),
    CONSTRAINT invoices_student_installment UNIQUE (student_id, session_id, installment_no),
    CONSTRAINT invoices_status CHECK (status IN ('DUE', 'PARTIAL', 'PAID', 'WAIVED', 'CANCELLED')),
    CONSTRAINT invoices_amounts_not_negative CHECK (
        gross_paise >= 0 AND concession_paise >= 0
            AND late_fee_paise >= 0 AND paid_paise >= 0
    ),
    CONSTRAINT invoices_concession_within_gross CHECK (concession_paise <= gross_paise)
);

CREATE INDEX invoices_student ON invoices (student_id, due_date);
CREATE INDEX invoices_due ON invoices (session_id, due_date, status);
CREATE INDEX invoices_outstanding ON invoices (status) WHERE status IN ('DUE', 'PARTIAL');

CREATE TABLE invoice_items (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   uuid   NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    fee_head_id  uuid   NOT NULL REFERENCES fee_heads (id),
    amount_paise bigint NOT NULL,
    concession_paise bigint NOT NULL DEFAULT 0,
    CONSTRAINT invoice_items_unique UNIQUE (invoice_id, fee_head_id),
    CONSTRAINT invoice_items_amount CHECK (amount_paise > 0 AND concession_paise >= 0)
);

CREATE INDEX invoice_items_invoice ON invoice_items (invoice_id);

CREATE TABLE receipts (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_no     text        NOT NULL,
    student_id     uuid        NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    session_id     uuid        NOT NULL REFERENCES academic_sessions (id),
    amount_paise   bigint      NOT NULL,
    mode           text        NOT NULL,
    cheque_no      text,
    cheque_date    date,
    bank_name      text,
    -- Filled by the payment gateway callback once one is connected. Empty for
    -- every cash and cheque receipt taken at the office counter.
    gateway_ref    text,
    gateway_status text,
    narration      text,
    received_at    timestamptz NOT NULL DEFAULT now(),
    received_by    uuid REFERENCES users (id),
    cancelled_at   timestamptz,
    cancelled_by   uuid REFERENCES users (id),
    cancel_reason  text,
    CONSTRAINT receipts_no_unique UNIQUE (receipt_no),
    CONSTRAINT receipts_amount CHECK (amount_paise > 0),
    CONSTRAINT receipts_mode CHECK (mode IN ('CASH', 'CHEQUE', 'UPI', 'CARD', 'NETBANKING', 'BANK_TRANSFER')),
    -- A cheque receipt without a cheque number is not a record of anything.
    CONSTRAINT receipts_cheque_details CHECK (
        mode <> 'CHEQUE' OR (cheque_no IS NOT NULL AND cheque_date IS NOT NULL)
    ),
    CONSTRAINT receipts_cancel_reason CHECK (
        cancelled_at IS NULL OR cancel_reason IS NOT NULL
    )
);

CREATE INDEX receipts_student ON receipts (student_id, received_at DESC);
CREATE INDEX receipts_day ON receipts (received_at DESC);
CREATE INDEX receipts_mode_day ON receipts (mode, received_at DESC);

-- One receipt can settle several installments, so the split is recorded rather
-- than inferred. The sum of a receipt's allocations equals its amount.
CREATE TABLE receipt_allocations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id   uuid   NOT NULL REFERENCES receipts (id) ON DELETE CASCADE,
    invoice_id   uuid   NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    amount_paise bigint NOT NULL,
    CONSTRAINT receipt_allocations_unique UNIQUE (receipt_id, invoice_id),
    CONSTRAINT receipt_allocations_amount CHECK (amount_paise > 0)
);

CREATE INDEX receipt_allocations_invoice ON receipt_allocations (invoice_id);
