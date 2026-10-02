"use client";

import * as api from "../../lib/api";
import { formatDateTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

const STATUS_TONES = { DRAFT: "neutral", QUEUED: "warn", SENT: "good", FAILED: "bad", CANCELLED: "neutral" };
const CHANNEL_LABELS = { SMS: "SMS", WHATSAPP: "WhatsApp", EMAIL: "Email", PUSH: "Push" };

export default function OutboxView({ lang, dict }) {
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/communication/outbox", { token, signal }),
  );
  const items = data?.items ?? data?.messages ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.outbox;

  const columns = [
    { key: "recipientName", header: lang === "hi" ? "प्राप्तकर्ता" : "Recipient", isPrimary: true },
    { key: "channel", header: lang === "hi" ? "माध्यम" : "Channel", render: (row) => <Badge tone="neutral">{CHANNEL_LABELS[row.channel] ?? row.channel}</Badge> },
    { key: "body", header: lang === "hi" ? "संदेश" : "Message", hideOnMobile: true, render: (row) => <span className="line-clamp-1">{row.body}</span> },
    { key: "status", header: lang === "hi" ? "स्थिति" : "Status", render: (row) => <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>{row.status}</Badge> },
    { key: "attempts", header: lang === "hi" ? "प्रयास" : "Tries", hideOnMobile: true, numeric: true },
    { key: "createdAt", header: lang === "hi" ? "समय" : "Time", hideOnMobile: true, render: (row) => formatDateTime(row.createdAt, lang) },
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
