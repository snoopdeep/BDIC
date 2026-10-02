"use client";

import * as api from "../../lib/api";
import { formatPercent } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function ResultsList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/exams/results", { token, signal }),
  );
  const items = data?.items ?? data?.results ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.results;

  const columns = [
    { key: "examName", header: lang === "hi" ? "परीक्षा" : "Exam", isPrimary: true, render: (row) => row.examName ?? row.exam ?? "—" },
    { key: "subject", header: lang === "hi" ? "विषय" : "Subject", render: (row) => row.subject ?? row.subjectName ?? "—" },
    { key: "marksObtained", header: lang === "hi" ? "प्राप्तांक" : "Marks", numeric: true, render: (row) => row.marksObtained ?? row.marks ?? "—" },
    { key: "maxMarks", header: lang === "hi" ? "पूर्णांक" : "Max", numeric: true, hideOnMobile: true },
    { key: "percentage", header: lang === "hi" ? "प्रतिशत" : "Percent", hideOnMobile: true, numeric: true, render: (row) => {
      const pct = row.percentage ?? (row.maxMarks > 0 ? ((row.marksObtained ?? 0) / row.maxMarks * 100) : null);
      return pct !== null && pct !== undefined ? formatPercent(pct, lang, 1) : "—";
    }},
    { key: "grade", header: lang === "hi" ? "श्रेणी" : "Grade", render: (row) => row.grade ? <Badge tone="neutral">{row.grade}</Badge> : "—" },
    { key: "result", header: lang === "hi" ? "परिणाम" : "Result", hideOnMobile: true, render: (row) => {
      const r = row.result ?? row.status;
      if (!r) return "—";
      return <Badge tone={r === "PASS" || r === "PASSED" ? "good" : r === "FAIL" || r === "FAILED" ? "bad" : "neutral"}>{r}</Badge>;
    }},
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
