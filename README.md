

# BDIC School Management System

A school management system for **Bhagwan Das Inter College, Ekauna, Chandauli** —
public bilingual website, online admissions, student and staff records,
timetable, attendance, homework, examinations, fees, and role-based portals for
the Principal, office, accounts staff, teachers, students and parents.

Everything works in **Hindi and English**, and is built mobile-first for a
mid-range Android phone on a patchy 4G connection.

The full feature list and the client proposal live in
[the proposal document](https://claude.ai/code/artifact/f32a3d1f-8e5a-497b-819f-29b2b8c5088b).

---

## Stack

| Part | Technology |
| --- | --- |
| API | Go 1.26, standard library HTTP, `pgx/v5` |
| Database | PostgreSQL 17, migrations embedded in the binary |
| Frontend | Next.js 16 App Router, React 19, Tailwind CSS v4 |
| Auth | HMAC-signed tokens, bcrypt at cost 12, server-side revocation |

No ORM, no Docker, no build step beyond `go` and `npm`. Dependencies are `pgx`
and `golang.org/x/crypto` on the backend, and Next, React and Tailwind on the
frontend.

---

## First-time setup

### 1. PostgreSQL

```bash
brew install postgresql@17
brew services start postgresql@17
```

### 2. Everything else

```bash
make setup
```

That fetches the Go modules, installs the npm packages, creates the `bdic`
database, and copies `.env.example` to `.env`.

### 3. Fill in `.env`

Two values are required before the API will start:

```bash
# Generate a signing key and paste it into AUTH_SIGNING_KEY
openssl rand -base64 48

# Check your Mac username for the database URL
whoami
```

Open `.env` and set:

- `AUTH_SIGNING_KEY` — the generated value. The API refuses to start without it.
- `DATABASE_URL` — replace `postgres@` with your own username, e.g.
  `postgres://deepakkumar.singh@localhost:5432/bdic?sslmode=disable`

Every other blank in `.env` is an external service the school has not opened an
account with yet. Leaving them blank is correct: see
[Credentials still needed](#credentials-still-needed).

---

## Running it

Two terminals:

```bash
make api      # http://localhost:8080
```

```bash
make web      # http://localhost:3000
```

The API applies any pending migrations on start-up and creates the first
administrator from `BOOTSTRAP_ADMIN_*`. There is no separate migration or seed
command to remember.

Open **http://localhost:3000** — you will be redirected to `/hi` or `/en`
depending on your browser's language.

Sign in at **http://localhost:3000/hi/login** with the bootstrap credentials
from your `.env`:

```text
admin@bdic.local / ChangeThisNow2026
```

That account is created with "must change password" set. Change it, create the
real Principal account, then **delete the `BOOTSTRAP_ADMIN_*` lines from
`.env`**. The API refuses to start in production while they are present.

Health check: <http://localhost:8080/health>

---

## Before committing

```bash
make check
```

Formats the Go code, runs `go vet`, runs the tests, and builds both sides.

---

## Project layout

```text
backend/
  cmd/api/                 Entry point: config, migrations, server, shutdown
  internal/
    api/                   HTTP handlers and the router
    audit/                 Append-only record of who changed what
    auth/                  Passwords, tokens, roles, middleware, rate limiting
    config/                Environment reading and validation
    db/                    Connection pool and the migration runner
    db/migrations/         Numbered .sql files, embedded in the binary
    httpx/                 JSON helpers and the bilingual error envelope
    store/                 Every SQL query in the application
frontend/
  app/[lang]/              All routes, under a language segment
    dictionaries/          en.json and hi.json — every string, both languages
    (site)/                The public website
    login/                 Sign in and password reset
    portal/                The signed-in portal
  components/              Shared React components
  lib/                     API client, session handling
  proxy.js                 Language detection and redirect
```

### Two things worth knowing about the layout

**There is no `app/layout.js`.** The root layout is `app/[lang]/layout.js`,
which is what makes `lang` available to every page below it.

**It is `proxy.js`, not `middleware.js`.** Next.js 16 renamed the convention.

---

## How the pieces fit

- The **database is the authority on permission**, not the screen. The
  navigation a teacher sees is built from their role as a courtesy; the API
  checks the role on every request and scopes every query, so typing another
  user's URL returns a 403, not their data.
- **Money is stored in integer paise.** There is no floating point anywhere in
  the fee tables, because a ledger has to balance to the rupee.
- **Nothing is hard-deleted.** A removed student, receipt or notice is marked
  removed and kept, so a mistake can be undone and a record can be produced.
- **A calendar date is text, not a timestamp.** Dates come out of Postgres
  already formatted `YYYY-MM-DD`, because a school date is a day, and carrying
  it as an instant is how it ends up a day out after a time-zone conversion.
- **Nothing is sent to anyone.** Every SMS, WhatsApp message and email the
  system would send is written to `outbound_messages` as a `DRAFT`. The school
  can read exactly what each parent would have received. A provider is
  connected later by filling in `.env`.

---

## Credentials still needed

None of these stop the system working locally. Each one switches on a feature
that needs an account in the school's own name.

| What | Where it goes | What it unlocks | Who has to open it |
| --- | --- | --- | --- |
| SMS provider API key + DLT registration | `SMS_*` in `.env` | Absence alerts and fee reminders by SMS | The school, with a telecom operator. Mandatory in India for transactional SMS |
| WhatsApp Business token | `WHATSAPP_*` | Notices over WhatsApp | The school, via Meta |
| SMTP credentials | `SMTP_*` | Email notices | The school, on its own email domain |
| Payment gateway keys | `PAYMENT_*` | Parents paying fees online | The school, with Razorpay / PayU / its bank, using the school's PAN and bank account |
| S3 bucket | `S3_*` | Uploads stored in the cloud rather than on local disk | At deployment |

Until then: the office can issue invoices, collect and allocate local payments,
and view the collection ledger. SMS/WhatsApp/email remain readable outbound
drafts until the school supplies provider credentials. Receipt PDF/print
templates and online-gateway callbacks are intentionally not represented as
complete.

---

## Current local implementation

The runnable local build now includes these operational API workflows, with
role-scoped access and audit records for their writes:

- Student and staff onboarding, with optional student/parent/staff portal
  accounts that must reset their temporary password.
- Student, parent, teacher, office, accounts, Principal and Super Admin views
  for attendance, timetable, homework, fees, receipts, certificates, and the
  activity log where appropriate.
- Same-day class attendance, an attendance submission summary, weekly
  timetables, dated substitutes, homework publishing, student submissions, and
  teacher feedback.
- Manual invoice issuance, atomic office receipt allocation, collection-day
  reporting, four serialised certificate types, and append-only audit events.

The portal currently uses a shared, responsive data screen for these modules.
It is deliberately functional but not the final role-specific office UI.

### Still required before calling it a production ERP

- Edit/detail screens for people, attendance correction approval, timetable
  editing, fee-plan generation, concessions, refunds/cancellations, printable
  receipts/certificates, and fuller export reports.
- Automated database integration and browser tests in CI. The local build has
  been smoke-tested end to end, but the checks are not yet a committed test
  suite.
- School-approved contacts, logo, photographs, policies, exact fee structure,
  grading rules, timetable, and historical records.
- HTTPS cookie authentication, backup/restore drills, monitoring, domain and
  email configuration, and the provider credentials listed above.

---

## Build stages

| Stage | Covers | State |
| --- | --- | --- |
| 1 | Database schema, migrations, auth, roles, audit log, bilingual shell, sign-in, portal frame | **Done** |
| 2 | The public website: all pages, notices, gallery, enquiry form, SEO | Next |
| 3 | Online admissions, from a parent's phone to an enrolled student | |
| 4 | Records, certificates, timetable, daily attendance | **Partially implemented locally** |
| 5 | Homework, study material, examinations, report cards | **Partially implemented locally** |
| 6 | Fee structures, concessions, collection, receipts, reports | **Partially implemented locally** |
| 7 | Notices, events, gallery consent, messaging, dashboards | |
| 8 | Seed data, tests, Hindi review, deployment | |

Each portal link now resolves to a loading, error, and empty-state-capable
screen. A role-specific data-entry interface is still required for the items
listed under the production gap above.

---

## Placeholder content

Every placeholder is marked `TODO school:` in the source and in
`backend/internal/db/migrations/0010_baseline_data.sql`. They are edited in the
app under School settings, not by changing the file. The ones that matter most:

- Affiliation number, UDISE code, phone numbers, email, Principal's name
- The logo and the brand colours (currently indigo placeholders)
- Photographs — the gallery, facilities and faculty all show nothing until real
  approved images are provided
- Period timings, the weekly period count per subject, and the elective
  subjects actually offered
- The grading scale and division boundaries, checked against BDIC's own report
  card
- The complete fee structure, installment dates, and the receipt number the
  series should continue from
