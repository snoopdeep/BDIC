"use client";

import * as api from "../../lib/api";
import { formatDate, formatDateTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

const STATUS_TONES = { SUBMITTED: "neutral", UNDER_REVIEW: "warn", SELECTED: "good", WAITLISTED: "warn", REJECTED: "bad", ENROLLED: "good" };
const STATUS_LABELS = {
  en: { SUBMITTED: "Submitted", UNDER_REVIEW: "Under review", SELECTED: "Selected", WAITLISTED: "Waitlisted", REJECTED: "Rejected", ENROLLED: "Enrolled" },
  hi: { SUBMITTED: "जमा", UNDER_REVIEW: "समीक्षाधीन", SELECTED: "चयनित", WAITLISTED: "प्रतीक्षा", REJECTED: "अस्वीकृत", ENROLLED: "नामांकित" },
};

export default function AdmissionsList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/admissions/my-applications", { token, signal }),
  );
  const items = data?.items ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.admissions;

  const columns = [
    { key: "applicationNo", header: lang === "hi" ? "आवेदन सं." : "App. No.", isPrimary: true },
    { key: "applicantName", header: lang === "hi" ? "आवेदक" : "Applicant" },
    { key: "fatherName", header: lang === "hi" ? "पिता का नाम" : "Father", hideOnMobile: true },
    { key: "guardianPhone", header: lang === "hi" ? "फ़ोन" : "Phone", hideOnMobile: true },
    { key: "submittedAt", header: lang === "hi" ? "तिथि" : "Date", hideOnMobile: true, render: (row) => formatDateTime(row.submittedAt, lang) },
    { key: "status", header: lang === "hi" ? "स्थिति" : "Status", render: (row) => <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>{STATUS_LABELS[lang]?.[row.status] ?? row.status}</Badge> },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title={title} />
      <Panel>
        {loading ? <LoadingBlock label={dict.common.loading} rows={5} /> : null}
        {!loading && error ? <div className="space-y-3"><Alert tone="bad">{errorMessage(error, dict, lang)}</Alert><Button variant="ghost" onClick={reload}>{dict.common.retry}</Button></div> : null}
        {!loading && !error ? <DataTable rows={items} columns={columns} caption={title} empty={<EmptyState title={dict.common.noResults} />} /> : null}
      </Panel>
    </div>
  );
}
