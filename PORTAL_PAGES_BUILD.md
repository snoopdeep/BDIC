# BDIC Portal Pages - Build Plan

## Overview

This is the target plan for dedicated, role-specific portal screens. It is not
a record that every component below has been built.

The checked-in application has a bilingual portal shell and a data-aware
`frontend/components/PortalSection.jsx` that renders the implemented module
APIs with loading, error, and empty states. The active operational endpoints
include people directories/onboarding, attendance, timetable, homework,
invoices/receipts/collection reporting, certificates, and audit events. The
per-screen component files below remain the next step for richer forms,
filters, print layouts, and role-specific workflows; do not describe them as
present until those files are added and browser-tested.

## Target architecture

### Page Files
Location: `/frontend/app/[lang]/portal/{section}/page.js`

Each page file:
- Is a server component that handles metadata generation
- Imports the dictionary for i18n
- Imports and renders the corresponding client component
- Passes `lang` and `dict` props to the component

### Component Files
Location: `/frontend/components/portal/{ComponentName}.jsx`

Each component:
- Is a client component (uses "use client")
- Manages its own data fetching and state
- Uses the session context to get user, token, and role info
- Calls the appropriate API endpoint
- Handles loading, error, and empty states
- Uses UI components from `../ui.jsx` for consistent styling

## Planned portal sections

### 1. Students (`/students`)
- **File**: `/components/portal/StudentsList.jsx`
- **API Endpoint**: `GET /api/v1/people/students`
- **Shows**: List of students with admission #, class, section, email, phone
- **Visible To**: Admins, principal, office staff, teachers, parents

### 2. Staff (`/staff`)
- **File**: `/components/portal/StaffList.jsx`
- **API Endpoint**: `GET /api/v1/people/staff`
- **Shows**: List of staff with role, department, email, phone
- **Visible To**: Admins, principal, office staff

### 3. Admissions (`/admissions`)
- **File**: `/components/portal/AdmissionsList.jsx`
- **API Endpoint**: `GET /api/v1/admissions/my-applications`
- **Shows**: Admission applications with status tracking
- **Visible To**: Admins, principal, office staff

### 4. Attendance (`/attendance`)
- **File**: `/components/portal/AttendanceView.jsx`
- **API Endpoint**: `GET /api/v1/teaching/attendance`
- **Shows**: Attendance records with status badges (present/absent/leave)
- **Visible To**: All signed-in users (teachers to mark, students/parents to view)

### 5. Timetable (`/timetable`)
- **File**: `/components/portal/TimetableView.jsx`
- **API Endpoint**: `GET /api/v1/teaching/my-classes`
- **Shows**: Class schedule with time, room, subject
- **Visible To**: Admins, principal, teachers, students, parents

### 6. Homework (`/homework`)
- **File**: `/components/portal/HomeworkList.jsx`
- **API Endpoint**: `GET /api/v1/teaching/homework`
- **Shows**: Homework assignments with due dates
- **Visible To**: Teachers, students, parents

### 7. Study Material (`/study-material`)
- **File**: `/components/portal/StudyMaterialList.jsx`
- **API Endpoint**: `GET /api/v1/teaching/study-material`
- **Shows**: Study materials by subject with upload dates
- **Visible To**: Teachers, students, parents

### 8. Exams (`/exams`)
- **File**: `/components/portal/ExamsList.jsx`
- **API Endpoint**: `GET /api/v1/exams/schedule`
- **Shows**: Exam schedule with dates, times, and room info
- **Visible To**: All users

### 9. Results (`/results`)
- **File**: `/components/portal/ResultsList.jsx`
- **API Endpoint**: `GET /api/v1/exams/results`
- **Shows**: Exam results with marks, percentage, and grade
- **Visible To**: Students, parents, admins, teachers

### 10. Report Card (`/report-card`)
- **File**: `/components/portal/ReportCardView.jsx`
- **API Endpoint**: `GET /api/v1/academics/records`
- **Shows**: Subject-wise marks and performance details
- **Visible To**: Students, parents

### 11. Fees (`/fees`)
- **File**: `/components/portal/FeesList.jsx`
- **API Endpoint**: `GET /api/v1/fees/invoices`
- **Shows**: Fee invoices with status (paid/pending/overdue)
- **Visible To**: Admins, accounts staff, students, parents

### 12. Receipts (`/receipts`)
- **File**: `/components/portal/ReceiptsList.jsx`
- **API Endpoint**: `GET /api/v1/fees/receipts`
- **Shows**: Payment receipts with download functionality
- **Visible To**: Admins, accounts staff, students, parents

### 13. Certificates (`/certificates`)
- **File**: `/components/portal/CertificatesList.jsx`
- **API Endpoint**: `GET /api/v1/academics/certificates`
- **Shows**: Available certificates with download option
- **Visible To**: Office staff, students, parents

### 14. Notices (`/notices`)
- **File**: `/components/portal/NoticesList.jsx`
- **API Endpoint**: `GET /api/v1/communication/notifications`
- **Shows**: School notices in panel format
- **Visible To**: All users

### 15. Messages (`/messages`)
- **File**: `/components/portal/MessagesList.jsx`
- **API Endpoint**: `GET /api/v1/communication/messages`
- **Shows**: Message inbox with read/unread status
- **Visible To**: Teachers, parents, office staff

### 16. Gallery (`/gallery`)
- **File**: `/components/portal/GalleryView.jsx`
- **API Endpoint**: `GET /api/v1/public/site/gallery`
- **Shows**: Photo gallery in a responsive grid layout
- **Visible To**: All users

### 17. Reports (`/reports`)
- **File**: `/components/portal/ReportsList.jsx`
- **API Endpoints**: `GET /api/v1/fees/reports` or `GET /api/v1/dashboard/reports`
- **Shows**: Fee reports and admin reports with download option
- **Visible To**: Accounts staff, admins, principal

### 18. Settings (`/settings`)
- **File**: `/components/portal/SettingsPanel.jsx`
- **API Endpoint**: `GET /api/v1/settings`, `PUT /api/v1/settings/{key}`
- **Shows**: School settings (editable by admins/principal only)
- **Visible To**: Admins, principal (read-only for others)

### 19. Audit Log (`/audit-log`)
- **File**: `/components/portal/AuditLogView.jsx`
- **API Endpoint**: `GET /api/v1/audit/logs`
- **Shows**: Activity log with timestamp, user, action, resource
- **Visible To**: Admins, principal

### 20. Outbox (`/outbox`)
- **File**: `/components/portal/OutboxView.jsx`
- **API Endpoint**: `GET /api/v1/communication/outbox`
- **Shows**: Sent messages with delivery status
- **Visible To**: Admins, office staff

## Target file structure

```
/frontend
├── app/[lang]/portal/
│   ├── students/page.js
│   ├── staff/page.js
│   ├── admissions/page.js
│   ├── attendance/page.js
│   ├── timetable/page.js
│   ├── homework/page.js
│   ├── study-material/page.js
│   ├── exams/page.js
│   ├── results/page.js
│   ├── report-card/page.js
│   ├── fees/page.js
│   ├── receipts/page.js
│   ├── certificates/page.js
│   ├── notices/page.js
│   ├── messages/page.js
│   ├── gallery/page.js
│   ├── reports/page.js
│   ├── settings/page.js
│   ├── audit-log/page.js
│   └── outbox/page.js
└── components/portal/
    ├── StudentsList.jsx
    ├── StaffList.jsx
    ├── AdmissionsList.jsx
    ├── AttendanceView.jsx
    ├── TimetableView.jsx
    ├── HomeworkList.jsx
    ├── StudyMaterialList.jsx
    ├── ExamsList.jsx
    ├── ResultsList.jsx
    ├── ReportCardView.jsx
    ├── FeesList.jsx
    ├── ReceiptsList.jsx
    ├── CertificatesList.jsx
    ├── NoticesList.jsx
    ├── MessagesList.jsx
    ├── GalleryView.jsx
    ├── ReportsList.jsx
    ├── SettingsPanel.jsx
    ├── AuditLogView.jsx
    └── OutboxView.jsx
```

## Required features for each planned screen

### Loading State
All components display a `<LoadingBlock>` with 5 rows while fetching data

### Empty State
Shows an `<EmptyState>` component when no data is available

### Error Handling
Displays an `<Alert>` with error details if the API request fails

### Data Display
- **Tables**: Uses the `<DataTable>` component which automatically adapts to mobile (card layout) and desktop (table layout)
- **Panels**: Uses the `<Panel>` component for structured layout
- **Badges**: Uses the `<Badge>` component for status indicators with color coding (good/warn/bad/neutral)

### Session Integration
All components:
- Import `useSession()` from `../../lib/session`
- Access `user`, `token`, and optionally `school` and `currentSession`
- Use the `token` for authenticated API calls
- Check `user.role` for role-specific behavior

### API Requests
All components:
- Import `request()` from `../../lib/api`
- Make requests with the bearer token
- Handle network errors gracefully
- Display appropriate error messages

## Internationalization

Each page file:
- Calls `getDictionary(lang)` to get localized strings
- Passes the dictionary to the component
- Components use dictionary strings from:
  - `dict.portal.sections.{sectionName}` for titles
  - `dict.common.*` for common button labels and messages
  - `dict.roles` for role names (when displayed)

## Build order

1. **Confirm each API contract**: The existing handlers are a starting point;
   each UI screen must use only endpoints that are present, authorised, and
   covered by an integration test.
2. **Create one screen at a time**: Add the route, client component,
   dictionaries, and role-specific data handling together.
3. **Add usable interactions**: Include features such as:
   - Search and filtering
   - Pagination
   - Sorting
   - Row actions (edit, delete, etc.)
   - Form submissions
   - Real-time updates
4. **Permission testing**: Verify role-based access control and row scoping
   with a test account for every role.
5. **Data-format validation**: Verify each API response against the UI's
   expected shape before relying on it.
6. **Performance and accessibility**: Add pagination, caching where safe,
   and a mobile-width review for each completed page.
7. **Testing**: Add unit and integration tests for each component and endpoint.

## Column Definitions

Each data table uses consistent column configuration:
- `key`: Property name in the data object
- `header`: Display label
- `isPrimary`: True for the primary column (shown as card heading on mobile)
- `hideOnMobile`: True to hide on small screens
- `align`: "left" (default), "center", or "right"
- `render`: Optional function to customize the cell value display

Example:
```javascript
const columns = [
  { key: "name", header: "Name", isPrimary: true },
  { key: "status", header: "Status", render: (row) => <Badge>{row.status}</Badge> },
  { key: "email", header: "Email", hideOnMobile: true },
];
```

## Notes

- The old placeholder page at `/app/[lang]/portal/[...section]/page.js` remains for backward compatibility if specific routes aren't matched
- All pages follow async server component patterns for Next.js 13+
- Components use client-side data fetching as the API is on a different port
- Error boundaries and fallbacks ensure graceful degradation
- The UI system uses Tailwind CSS for responsive design
