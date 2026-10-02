# Conventions

How this codebase is written. Follow these and a new module will look like the
existing ones; ignore them and it will not.

---

## Backend

### Layering

```
cmd/api/main.go        Wiring only. No business logic.
internal/api/          HTTP handlers. Parse, authorise, call the store, respond.
internal/store/        Every SQL query in the application. No HTTP types here.
internal/auth/         Passwords, tokens, roles, middleware.
internal/audit/        Append-only record of who changed what.
internal/httpx/        JSON helpers and the bilingual error envelope.
internal/files/        File storage behind an interface.
internal/db/           Pool and the embedded migration runner.
```

A handler never writes SQL. A store method never touches `http.Request`.

### One file per module, on both sides

A module owns exactly these files and edits no others:

```
internal/store/<module>.go          Its queries and its types
internal/api/<module>_handlers.go   Its handlers and its register function
```

`internal/api/server.go` already calls `s.register<Module>Routes(mux)`. Define
that function in your handlers file. Do not edit `server.go`.

### Route registration

```go
func (s *Server) registerFeeRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/fees/invoices",
		s.restricted(s.handleListInvoices, auth.RoleAccounts, auth.RolePrincipal, auth.RoleSuperAdmin))
	mux.Handle("POST /api/v1/fees/receipts",
		s.restricted(s.handleCreateReceipt, auth.RoleAccounts, auth.RoleSuperAdmin))
	mux.Handle("GET /api/v1/fees/my-dues", s.signedIn(s.handleMyDues))
}
```

- `s.signedIn(handler)` — any authenticated, active user.
- `s.restricted(handler, roles...)` — only those roles.
- `mux.HandleFunc(...)` — open, no token. Use only for genuinely public data.

Go 1.22+ `ServeMux` matches on method and path, with `{param}` wildcards read
via `r.PathValue("param")`. There is no third-party router.

**The role check is the coarse gate. Row scoping happens in the query.** A
teacher reaching `/attendance` is a role check; a teacher only seeing *their
own* sections is a `WHERE` clause. Both are required.

### Handler shape

```go
func (s *Server) handleCreateThing(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var input createThingRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	thing, err := s.store.CreateThing(r.Context(), store.NewThing{ /* ... */ })
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "thing",
		EntityID:   thing.ID,
		Summary:    "Created " + thing.Name,
		AfterState: thing,
	})

	httpx.JSON(w, http.StatusCreated, thing)
}
```

Helpers already available in `internal/api/helpers.go`:

| Helper | Does |
| --- | --- |
| `pathUUID(r, "id")` | Reads and validates a path parameter as a UUID |
| `queryUUID(r, "classId")` | Optional UUID query parameter; `""` means no filter |
| `paginate(r)` | Returns `limit, offset` from the query string |
| `s.currentSessionID(r)` | `?sessionId=` or the session marked current |
| `storeError(err)` | Maps `store.ErrNotFound` / `ErrConflict` to the right status |
| `contextWithTimeout(r, d)` | Bounded context for a slow query |

### Errors

Never `http.Error`. Always `httpx.Fail(w, r, err)` with one of:

```go
httpx.ErrBadRequest("Enter a due date.", "नियत तिथि दर्ज करें।")
httpx.ErrNotFound()
httpx.ErrForbidden()
httpx.ErrConflict("That receipt number already exists.", "यह रसीद संख्या पहले से है।")
httpx.ErrInternal().WithInternal(err)   // the cause is logged, never sent
```

Add `.WithField("dueDate", "Required")` so a form can highlight the one input
that is wrong. **Every user-facing message is written in both languages, in
place.** There is no translation layer on the backend.

### Store shape

```go
// Thing is one row as the API returns it.
type Thing struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Store) ListThings(ctx context.Context, sessionID string, limit, offset int) ([]Thing, int, error) {
	// count first, then the page — a list endpoint returns both
}
```

Rules, all of which the existing queries follow:

- **Bind parameters only.** No string concatenation of user input into SQL,
  ever. Where a query is composed from constants, keep the select list and the
  `FROM` as separate constants (see `store/users.go`) so a column cannot land
  after the `FROM`.
- **Cast UUID parameters explicitly**: `WHERE id = $1::uuid`. For an optional
  one, `($2 = '' OR x = NULLIF($2, '')::uuid)`.
- **Nullable columns**: wrap in `COALESCE(col, '')` and scan into `string`, or
  scan into `*string`. Never scan a nullable column into a bare `string`.
- **Dates**: select as text — `to_char(due_date, 'YYYY-MM-DD')` — and carry
  them as Go `string`. A school date is a calendar day; carrying it as a
  timestamp is how it ends up a day out after a time-zone conversion.
- **Timestamps**: `to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')`.
- **Money**: integer paise, in `int64`. No floats anywhere near a fee.
- **Slices are never nil**: initialise `items := []Thing{}` so the JSON is `[]`
  and the frontend never guards against `null`.
- **Multi-row writes go in `s.InTx(ctx, func(tx pgx.Tx) error { … })`.**
- **Issue numbers with `store.AllocateNumber(ctx, tx, store.SeriesReceiptNo, period)`.**
  Never a sequence: sequences leak numbers on rollback and a fee book with gaps
  is a fee book an auditor asks about.

### Audit

Record anything the school could be asked to account for: a mark entered or
changed, a receipt taken or cancelled, attendance corrected after the day, a
concession approved, a result published, a photograph approved, an admission
decided. Use the `audit.Action*` constants. Reads are not audited; exports are.

### The schema is already built

All tables exist. **Read `internal/db/migrations/*.sql` before writing a
query** and use the columns that are there. Only add a migration if something
is genuinely missing, and then as a new numbered file — never by editing an
applied one.

### Nothing is sent, and no money moves

Any SMS, WhatsApp message or email goes in as a `DRAFT` row via
`s.store.QueueOutbound(...)`. No provider is connected. The fee module works
fully for office collection; the online gateway is not wired, and
`PAYMENT_*` in `.env` is blank on purpose. Mark such seams with
`// TODO school:` and a note on what is needed.

---

## Frontend

### Next.js 16, not 14 or 15

- `params` and `searchParams` are **Promises**. `const { lang } = await params;`
- Client components cannot be `async`. Read params with React's `use()`.
- It is `proxy.js`, not `middleware.js`.
- There is no `app/layout.js`. The root layout is `app/[lang]/layout.js`.
- Caching is opt-in. `lib/api.js` already chooses correctly per request.
- Read `node_modules/next/dist/docs/` before reaching for an API you are
  unsure about.

### Where things go

```
app/[lang]/(site)/<page>/page.js       A public page
app/[lang]/portal/<section>/page.js    A portal screen (server component)
components/<module>/<Name>.jsx         That module's components
app/[lang]/dictionaries/<module>.en.json   Its English strings
app/[lang]/dictionaries/<module>.hi.json   Its Hindi strings
```

A portal screen is a thin server component that loads the dictionary and hands
it to a client component:

```jsx
// app/[lang]/portal/fees/page.js
import FeesScreen from "../../../../components/fees/FeesScreen";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.fees };
}

export default async function FeesPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <FeesScreen lang={lang} dict={dict} />;
}
```

Adding `app/[lang]/portal/fees/page.js` automatically retires the
"coming soon" placeholder for that path: a named segment beats the catch-all.

### Dictionaries

Your module owns `<module>.en.json` and `<module>.hi.json`, each a single
top-level object keyed by the module name:

```json
{ "fees": { "title": "Fees", "collect": "Collect fee", "dues": "Outstanding dues" } }
```

Read them as `dict.fees.collect`. **Both files must have identical key sets** —
there is a checker for this. Do not touch `en.json` or `hi.json`; those are the
core strings, shared.

Hindi must read like Hindi, not like translated English. Use the school's own
vocabulary: शुल्क for fee, उपस्थिति for attendance, गृहकार्य for homework,
अंकपत्र for report card, अनुक्रमांक for admission number, कक्षा for class,
सत्र for session, अभिभावक for parent, प्रधानाचार्य for principal.

### Data and state

From `lib/portal.js`:

```jsx
"use client";
import { useApiData, useApiAction, errorMessage, useDebounced } from "../../lib/portal";

const { data, error, loading, reload } = useApiData(
  (token, signal) => api.request("/api/v1/fees/invoices", { token, signal }),
  { deps: [classId] },
);

const save = useApiAction((token, input) =>
  api.request("/api/v1/fees/receipts", { method: "POST", token, body: input }),
);
// save.run(input) → { ok, result }; save.busy; save.error; save.fieldErrors
```

Add your endpoints as named functions in `lib/api.js` only if they are used by
more than one screen; otherwise call `api.request` directly.

### UI

Everything comes from `components/ui.jsx`. Do not hand-roll a button or a
table.

`PageHeader` · `Panel` · `StatCard` · `Alert` · `LoadingBlock` · `EmptyState`
· `Badge` · `Button` · `Field` · `TextInput` · `TextArea` · `Select` ·
`Checkbox` · `DataTable` · `Pagination` · `Modal` · `ConfirmButton` ·
`FilterBar` · `FilterSelect` · `SearchInput` · `Printable`

`Field` takes a render function that supplies `id`, `aria-invalid`,
`aria-describedby` and the control class:

```jsx
<Field label={dict.fees.amount} error={save.fieldErrors.amount} required>
  {(props) => (
    <TextInput {...props} inputMode="decimal" value={amount}
      onChange={(event) => setAmount(event.target.value)} />
  )}
</Field>
```

`DataTable` turns into cards on a phone. Mark one column `isPrimary` (it
becomes the card heading) and any column that should not appear on a phone
`hideOnMobile`. Numeric columns get `numeric: true` for tabular figures.

Formatting comes from `lib/format.js`: `formatMoney(paise)`,
`paiseFromInput(text)`, `formatDate(iso, lang)`, `formatDateTime`,
`formatTime`, `formatPercent`, `formatNumber`, `todayISO()`,
`localised(record, "name", lang)`, `daysUntil(iso)`. Never use
`new Date("2026-08-15")` on a date-only string — it is parsed as UTC and
renders as the day before west of Greenwich.

### Design rules

- **44px minimum tap targets.** Buttons in `ui.jsx` already meet this.
- **Every state handled**: loading, empty, error, and success. A screen that
  only renders the happy path is not finished.
- **Colour never carries meaning alone.** A status is a word, optionally in a
  coloured `Badge`.
- **Mobile first.** Check every screen at 375px wide. Nothing scrolls sideways
  except a table inside its own container.
- **Printable where it matters**: the attendance register, fee receipts, report
  cards. Use `.no-print` on chrome and `.print-break-after` between pages;
  `globals.css` has the print styles.
- Never write a colour literal. Use the tokens: `brand-*`, `accent-*`,
  `good-*`, `warn-*`, `bad-*`, `zinc-*`.
- Tailwind v4: it is `backdrop-blur-sm`, `shadow-sm`, `outline-hidden` — the
  bare v3 names generate nothing.

### Placeholder content

Mark anything the school must supply with `TODO school:` and a note on what is
needed. Use obviously-sample data, never invented facts that could be mistaken
for real ones.
