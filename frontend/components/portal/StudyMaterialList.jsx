"use client";

import * as api from "../../lib/api";
import { formatDate } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function StudyMaterialList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/teaching/study-material", { token, signal }),
  );
  const items = data?.items ?? data?.materials ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.studyMaterial;

  const columns = [
    { key: "title", header: lang === "hi" ? "शीर्षक" : "Title", isPrimary: true, render: (row) => row.title ?? row.name ?? "—" },
    { key: "subjectName", header: lang === "hi" ? "विषय" : "Subject", render: (row) => row.subjectName ?? row.subject ?? "—" },
    { key: "className", header: lang === "hi" ? "कक्षा" : "Class", hideOnMobile: true, render: (row) => row.className ?? "—" },
    { key: "teacherName", header: lang === "hi" ? "शिक्षक" : "Teacher", hideOnMobile: true, render: (row) => row.teacherName ?? row.uploadedBy ?? "—" },
    { key: "createdAt", header: lang === "hi" ? "तिथि" : "Date", hideOnMobile: true, render: (row) => row.createdAt ? formatDate(row.createdAt?.slice?.(0, 10), lang, { short: true }) : "—" },
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
