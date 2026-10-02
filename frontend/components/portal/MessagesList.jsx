"use client";

import * as api from "../../lib/api";
import { formatDateTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function MessagesList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/communication/messages", { token, signal }),
  );
  const items = data?.items ?? data?.threads ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.messages;

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      {loading ? <Panel><LoadingBlock label={dict.common.loading} rows={5} /></Panel> : null}
      {!loading && error ? <Panel><div className="space-y-3"><Alert tone="bad">{errorMessage(error, dict, lang)}</Alert><Button variant="ghost" onClick={reload}>{dict.common.retry}</Button></div></Panel> : null}

      {!loading && !error && items.length === 0 ? <Panel><EmptyState title={dict.common.noResults} /></Panel> : null}

      {!loading && !error && items.length > 0 ? (
        <div className="space-y-3">
          {items.map((thread) => (
            <Panel key={thread.id}>
              <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-base font-semibold text-brand-900">
                      {thread.subject ?? `${thread.studentName ?? ""} – ${thread.staffName ?? ""}`}
                    </h3>
                    {thread.unreadCount > 0 ? (
                      <Badge tone="warn">{thread.unreadCount} {lang === "hi" ? "अपठित" : "unread"}</Badge>
                    ) : null}
                    <Badge tone="neutral">{thread.status}</Badge>
                  </div>
                  <div className="mt-1 flex flex-wrap gap-x-4 text-sm text-zinc-600">
                    {thread.studentName ? <span>{lang === "hi" ? "विद्यार्थी" : "Student"}: {thread.studentName}</span> : null}
                    {thread.staffName ? <span>{lang === "hi" ? "शिक्षक" : "Staff"}: {thread.staffName}</span> : null}
                    {thread.guardianName ? <span>{lang === "hi" ? "अभिभावक" : "Guardian"}: {thread.guardianName}</span> : null}
                  </div>
                  <p className="mt-1 text-xs text-zinc-500">{formatDateTime(thread.updatedAt, lang)}</p>
                </div>
              </div>
            </Panel>
          ))}
        </div>
      ) : null}
    </div>
  );
}
