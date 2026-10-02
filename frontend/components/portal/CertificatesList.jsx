"use client";

import * as api from "../../lib/api";
import { formatDate } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

const KIND_LABELS = {
  en: { BONAFIDE: "Bonafide", CHARACTER: "Character", TRANSFER: "Transfer", ATTENDANCE: "Attendance" },
  hi: { BONAFIDE: "बोनाफाइड", CHARACTER: "चरित्र", TRANSFER: "स्थानांतरण", ATTENDANCE: "उपस्थिति" },
};

export default function CertificatesList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/certificates", { token, signal }),
  );
  const items = data?.items ?? [];
  const title = dict.portal.sections.certificates;

  const columns = [
    { key: "serialNo", header: lang === "hi" ? "क्रमांक" : "Serial No.", isPrimary: true },
    { key: "studentName", header: lang === "hi" ? "विद्यार्थी" : "Student" },
    { key: "admissionNo", header: lang === "hi" ? "अनुक्रमांक" : "Adm. No.", hideOnMobile: true },
    { key: "kind", header: lang === "hi" ? "प्रकार" : "Type", render: (row) => <Badge tone="neutral">{KIND_LABELS[lang]?.[row.kind] ?? row.kind}</Badge> },
    { key: "issuedOn", header: lang === "hi" ? "जारी तिथि" : "Issued on", render: (row) => formatDate(row.issuedOn, lang, { short: true }) },
    { key: "cancelled", header: lang === "hi" ? "स्थिति" : "Status", hideOnMobile: true, render: (row) => row.cancelled ? <Badge tone="bad">{lang === "hi" ? "रद्द" : "Cancelled"}</Badge> : <Badge tone="good">{lang === "hi" ? "मान्य" : "Active"}</Badge> },
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
