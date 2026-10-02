"use client";

import * as api from "../../lib/api";
import { formatDateTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function NoticesList({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/communication/notifications", { token, signal }),
  );
  const items = data?.items ?? data?.notifications ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.notices;

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      {loading ? <Panel><LoadingBlock label={dict.common.loading} rows={5} /></Panel> : null}
      {!loading && error ? <Panel><div className="space-y-3"><Alert tone="bad">{errorMessage(error, dict, lang)}</Alert><Button variant="ghost" onClick={reload}>{dict.common.retry}</Button></div></Panel> : null}

      {!loading && !error && items.length === 0 ? <Panel><EmptyState title={dict.common.noResults} /></Panel> : null}

      {!loading && !error && items.length > 0 ? (
        <div className="space-y-3">
          {items.map((notice) => (
            <Panel key={notice.id}>
              <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-base font-semibold text-brand-900">{notice.title}</h3>
                    {notice.messageType ? <Badge tone="neutral">{notice.messageType}</Badge> : null}
                    {!notice.readAt ? <Badge tone="warn">{lang === "hi" ? "नया" : "New"}</Badge> : null}
                  </div>
                  {notice.body ? <p className="mt-1 text-sm text-zinc-700 line-clamp-3">{notice.body}</p> : null}
                  <p className="mt-2 text-xs text-zinc-500">{formatDateTime(notice.publishedAt, lang)}</p>
                </div>
                {notice.requiresAck && !notice.acknowledgedAt ? (
                  <Badge tone="bad">{lang === "hi" ? "पावती आवश्यक" : "Ack required"}</Badge>
                ) : null}
              </div>
            </Panel>
          ))}
        </div>
      ) : null}
    </div>
  );
}
