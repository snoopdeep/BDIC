"use client";

import { useMemo } from "react";

import * as api from "../lib/api";
import { errorMessage, useApiData } from "../lib/portal";
import { Alert, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "./ui";

// Each section owns the API response it displays. A section with no records is
// still a finished screen: it shows a proper empty state rather than a generic
// "coming soon" page. More advanced create/edit flows are added alongside the
// matching API write endpoint, not hidden behind a pretend button.
const SECTIONS = {
  admissions: { endpoint: "/api/v1/admissions/my-applications", keys: ["items"] },
	attendance: { endpoint: "/api/v1/attendance", keys: ["items"] },
	timetable: { endpoint: "/api/v1/timetable", keys: ["items"] },
	homework: { endpoint: "/api/v1/homework", keys: ["items"] },
	certificates: { endpoint: "/api/v1/certificates", keys: ["items"] },
	auditLog: { endpoint: "/api/v1/audit-log", keys: ["items"] },
	students: { endpoint: "/api/v1/people/students", keys: ["items"] },
	staff: { endpoint: "/api/v1/people/staff", keys: ["items"] },
  fees: { endpoint: "/api/v1/fees/invoices", keys: ["items"] },
  receipts: { endpoint: "/api/v1/fees/receipts", keys: ["items"] },
  studyMaterial: { endpoint: "/api/v1/teaching/study-material", keys: ["items", "materials"] },
  exams: { endpoint: "/api/v1/exams/schedule", keys: ["items", "schedule", "exams"] },
  results: { endpoint: "/api/v1/exams/results", keys: ["items", "results"] },
  reportCard: { endpoint: "/api/v1/academics/records", keys: ["items", "records"] },
  notices: { endpoint: "/api/v1/communication/notifications", keys: ["notifications", "items"] },
  messages: { endpoint: "/api/v1/communication/messages", keys: ["threads", "items"] },
  gallery: { endpoint: "/api/v1/public/site/gallery", keys: ["images", "items"] },
  reports: { endpoint: "/api/v1/fees/collection-report", keys: ["items"] },
  settings: { endpoint: "/api/v1/settings", keys: ["items", "settings"] },
  outbox: { endpoint: "/api/v1/communication/outbox", keys: ["messages", "items"] },
};

export default function PortalSection({ section, lang, dict }) {
  const config = SECTIONS[section];
  const title = dict.portal.sections[section] ?? dict.portal.title;
  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      config ? api.request(config.endpoint, { token, signal }) : Promise.resolve(null),
    { skip: !config },
  );

  const rows = useMemo(() => extractRows(data, config?.keys), [data, config]);

  return (
    <div className="space-y-6">
      <PageHeader title={title} subtitle={dict.common.noResults} />

      {config ? (
        <Panel title={title}>
          {loading ? <LoadingBlock label={dict.common.loading} rows={5} /> : null}
          {!loading && error ? (
            <div className="space-y-3">
              <Alert tone="bad">{errorMessage(error, dict, lang)}</Alert>
              <button type="button" onClick={reload} className="text-sm font-semibold text-brand-700">
                {dict.common.retry}
              </button>
            </div>
          ) : null}
          {!loading && !error ? (
            <DataTable
              rows={rows}
              columns={columnsFor(rows, lang)}
              caption={title}
              empty={<EmptyState title={dict.common.noResults} />}
            />
          ) : null}
        </Panel>
      ) : (
        <Panel title={title}>
          <EmptyState title={dict.common.noResults} />
        </Panel>
      )}
    </div>
  );
}

function extractRows(data, keys = []) {
  if (Array.isArray(data)) return data;
  if (!data || typeof data !== "object") return [];
  for (const key of keys) {
    if (Array.isArray(data[key])) return data[key];
  }
  // A summary response is presented as one readable record.
  return Object.keys(data).length ? [data] : [];
}

function columnsFor(rows, lang) {
  const sample = rows[0] ?? {};
  const keys = Object.keys(sample)
    .filter((key) => !key.endsWith("Id") && key !== "id")
    .slice(0, 5);
  const visible = keys.length ? keys : ["id"];

  return visible.map((key, index) => ({
    key,
    header: readableKey(key, lang),
    isPrimary: index === 0,
    hideOnMobile: index > 2,
    render: (row) => readableValue(row[key], lang),
  }));
}

function readableKey(key, lang) {
  const labels = {
    en: { status: "Status", date: "Date", title: "Title", name: "Name", body: "Details", count: "Count" },
    hi: { status: "स्थिति", date: "दिनांक", title: "शीर्षक", name: "नाम", body: "विवरण", count: "संख्या" },
  };
  return labels[lang]?.[key] ?? key.replace(/([A-Z])/g, " $1").replace(/^./, (letter) => letter.toUpperCase());
}

function readableValue(value, lang) {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "boolean") return value ? (lang === "hi" ? "हाँ" : "Yes") : (lang === "hi" ? "नहीं" : "No");
  if (typeof value === "object") return Array.isArray(value) ? String(value.length) : "—";
  return String(value);
}
