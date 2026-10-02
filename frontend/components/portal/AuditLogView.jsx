"use client";

import { useState } from "react";

import * as api from "../../lib/api";
import { formatDateTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import {
  Alert,
  Badge,
  Button,
  DataTable,
  EmptyState,
  LoadingBlock,
  PageHeader,
  Panel,
} from "../ui";

const PAGE_SIZE = 25;

const ACTION_TONES = {
  CREATE: "good",
  UPDATE: "neutral",
  DELETE: "bad",
  FEE_COLLECT: "good",
  APPROVE: "good",
  REJECT: "bad",
  PUBLISH: "good",
  EXPORT: "neutral",
};

export default function AuditLogView({ lang, dict }) {
  const [page, setPage] = useState(0);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/audit-log?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const items = data?.items ?? [];
  const title = dict.portal.sections.auditLog;

  const columns = [
    {
      key: "summary",
      header: lang === "hi" ? "विवरण" : "Description",
      isPrimary: true,
    },
    {
      key: "actorName",
      header: lang === "hi" ? "उपयोगकर्ता" : "User",
      render: (row) => (
        <span>
          {row.actorName}
          {row.actorRole ? (
            <>
              {" "}
              <Badge tone="neutral">
                {dict.roles[row.actorRole] ?? row.actorRole}
              </Badge>
            </>
          ) : null}
        </span>
      ),
    },
    {
      key: "action",
      header: lang === "hi" ? "कार्य" : "Action",
      render: (row) => (
        <Badge tone={ACTION_TONES[row.action] ?? "neutral"}>
          {row.action}
        </Badge>
      ),
    },
    {
      key: "entityType",
      header: lang === "hi" ? "प्रकार" : "Type",
      hideOnMobile: true,
    },
    {
      key: "createdAt",
      header: lang === "hi" ? "समय" : "Time",
      hideOnMobile: true,
      render: (row) => formatDateTime(row.createdAt, lang),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      <Panel>
        {loading ? <LoadingBlock label={dict.common.loading} rows={5} /> : null}

        {!loading && error ? (
          <div className="space-y-3">
            <Alert tone="bad">{errorMessage(error, dict, lang)}</Alert>
            <Button variant="ghost" onClick={reload}>
              {dict.common.retry}
            </Button>
          </div>
        ) : null}

        {!loading && !error ? (
          <>
            <DataTable
              rows={items}
              columns={columns}
              caption={title}
              empty={<EmptyState title={dict.common.noResults} />}
            />
            {items.length >= PAGE_SIZE ? (
              <div className="mt-4 flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                >
                  {lang === "hi" ? "पिछला" : "Previous"}
                </Button>
                <span className="text-sm text-zinc-600">
                  {lang === "hi" ? "पृष्ठ" : "Page"} {page + 1}
                </span>
                <Button
                  variant="outline"
                  disabled={items.length < PAGE_SIZE}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {lang === "hi" ? "अगला" : "Next"}
                </Button>
              </div>
            ) : null}
          </>
        ) : null}
      </Panel>
    </div>
  );
}
