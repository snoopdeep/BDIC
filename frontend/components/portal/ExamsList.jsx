"use client";

import * as api from "../../lib/api";
import { formatDate } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function ExamsList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/exams/schedule", { token, signal }),
  );
  const items = data?.items ?? data?.schedule ?? data?.exams ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.exams;

  const columns = [
    { key: "examName", header: lang === "hi" ? "परीक्षा" : "Exam", isPrimary: true },
    { key: "subject", header: lang === "hi" ? "विषय" : "Subject" },
    { key: "examDate", header: lang === "hi" ? "तिथि" : "Date", render: (row) => formatDate(row.examDate, lang, { short: true, withWeekday: true }) },
    { key: "startTime", header: lang === "hi" ? "समय" : "Time", hideOnMobile: true, render: (row) => row.startTime ? `${row.startTime}${row.endTime ? ` – ${row.endTime}` : ""}` : "—" },
    { key: "maxMarks", header: lang === "hi" ? "पूर्णांक" : "Max marks", hideOnMobile: true, numeric: true },
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
