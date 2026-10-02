"use client";

import { useState } from "react";
import * as api from "../../lib/api";
import { formatDate, daysUntil } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { useSession } from "../../lib/session";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

const PAGE_SIZE = 25;
const STATUS_TONES = { DRAFT: "neutral", PUBLISHED: "good", CLOSED: "warn" };

export default function HomeworkList({ lang, dict }) {
  const { user } = useSession();
  const [page, setPage] = useState(0);
  const { data, error, loading, reload } = useApiData(
    (token, signal) => api.request(`/api/v1/homework?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`, { token, signal }),
    { deps: [page] },
  );
  const items = data?.items ?? [];
  const title = dict.portal.sections.homework;
  const isTeacher = user?.role === "TEACHER";

  const columns = [
    { key: "title", header: lang === "hi" ? "शीर्षक" : "Title", isPrimary: true },
    { key: "subjectName", header: lang === "hi" ? "विषय" : "Subject" },
    ...(!isTeacher ? [{ key: "teacherName", header: lang === "hi" ? "शिक्षक" : "Teacher", hideOnMobile: true }] : []),
    ...(isTeacher ? [{ key: "className", header: lang === "hi" ? "कक्षा" : "Class", render: (row) => `${row.className}${row.sectionName ? ` – ${row.sectionName}` : ""}` }] : []),
    { key: "dueDate", header: lang === "hi" ? "अंतिम तिथि" : "Due date", render: (row) => {
      const days = daysUntil(row.dueDate);
      return <span>{formatDate(row.dueDate, lang, { short: true })}{days !== null && days < 0 ? <span className="ml-1 text-xs text-bad-600">({lang === "hi" ? "बीत चुकी" : "past due"})</span> : days === 0 ? <span className="ml-1 text-xs text-warn-600">({lang === "hi" ? "आज" : "today"})</span> : null}</span>;
    }},
    { key: "status", header: lang === "hi" ? "स्थिति" : "Status", hideOnMobile: true, render: (row) => <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>{row.status}</Badge> },
    { key: "attachmentCount", header: lang === "hi" ? "संलग्न" : "Files", hideOnMobile: true, numeric: true, render: (row) => row.attachmentCount || "—" },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title={title} />
      <Panel>
        {loading ? <LoadingBlock label={dict.common.loading} rows={5} /> : null}
        {!loading && error ? <div className="space-y-3"><Alert tone="bad">{errorMessage(error, dict, lang)}</Alert><Button variant="ghost" onClick={reload}>{dict.common.retry}</Button></div> : null}
        {!loading && !error ? (
          <>
            <DataTable rows={items} columns={columns} caption={title} empty={<EmptyState title={dict.common.noResults} />} />
            {items.length >= PAGE_SIZE ? (
              <div className="mt-4 flex items-center justify-end gap-2">
                <Button variant="outline" disabled={page === 0} onClick={() => setPage((p) => Math.max(0, p - 1))}>{lang === "hi" ? "पिछला" : "Previous"}</Button>
                <span className="text-sm text-zinc-600">{lang === "hi" ? "पृष्ठ" : "Page"} {page + 1}</span>
                <Button variant="outline" disabled={items.length < PAGE_SIZE} onClick={() => setPage((p) => p + 1)}>{lang === "hi" ? "अगला" : "Next"}</Button>
              </div>
            ) : null}
          </>
        ) : null}
      </Panel>
    </div>
  );
}
