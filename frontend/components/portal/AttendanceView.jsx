"use client";

import { useState } from "react";

import * as api from "../../lib/api";
import { formatDate } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { useSession } from "../../lib/session";
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

const STATUS_TONES = {
  PRESENT: "good",
  ABSENT: "bad",
  LATE: "warn",
  LEAVE: "neutral",
  MEDICAL: "neutral",
  HALF_DAY: "warn",
};

const STATUS_LABELS = {
  en: {
    PRESENT: "Present",
    ABSENT: "Absent",
    LATE: "Late",
    LEAVE: "Leave",
    MEDICAL: "Medical",
    HALF_DAY: "Half day",
  },
  hi: {
    PRESENT: "उपस्थित",
    ABSENT: "अनुपस्थित",
    LATE: "देरी",
    LEAVE: "छुट्टी",
    MEDICAL: "चिकित्सा",
    HALF_DAY: "आधा दिन",
  },
};

export default function AttendanceView({ lang, dict }) {
  const { user } = useSession();
  const [page, setPage] = useState(0);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/attendance?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const items = data?.items ?? [];
  const title = dict.portal.sections.attendance;

  // Students and parents see their own attendance; staff see all
  const isPersonalView =
    user?.role === "STUDENT" || user?.role === "PARENT";

  const columns = [
    {
      key: "date",
      header: lang === "hi" ? "दिनांक" : "Date",
      isPrimary: isPersonalView,
      render: (row) => formatDate(row.date, lang),
    },
    ...(isPersonalView
      ? []
      : [
          {
            key: "studentName",
            header: lang === "hi" ? "विद्यार्थी" : "Student",
            isPrimary: true,
          },
          {
            key: "admissionNo",
            header: lang === "hi" ? "अनुक्रमांक" : "Adm. No.",
            hideOnMobile: true,
          },
        ]),
    {
      key: "sectionName",
      header: lang === "hi" ? "सेक्शन" : "Section",
      hideOnMobile: true,
      render: (row) => row.sectionName || "—",
    },
    {
      key: "status",
      header: lang === "hi" ? "स्थिति" : "Status",
      render: (row) => (
        <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>
          {STATUS_LABELS[lang]?.[row.status] ?? row.status}
        </Badge>
      ),
    },
    {
      key: "note",
      header: lang === "hi" ? "टिप्पणी" : "Note",
      hideOnMobile: true,
      render: (row) => row.note || "—",
    },
  ];

  // Summary counts for personal view
  const summary = isPersonalView
    ? items.reduce(
        (acc, item) => {
          acc[item.status] = (acc[item.status] || 0) + 1;
          return acc;
        },
        {},
      )
    : null;

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      {/* Summary badges for student/parent view */}
      {summary && Object.keys(summary).length > 0 ? (
        <div className="flex flex-wrap gap-2">
          {Object.entries(summary).map(([status, count]) => (
            <div
              key={status}
              className="card flex items-center gap-2 px-3 py-2"
            >
              <Badge tone={STATUS_TONES[status] ?? "neutral"}>
                {STATUS_LABELS[lang]?.[status] ?? status}
              </Badge>
              <span className="text-sm font-semibold tabular text-brand-900">
                {count}
              </span>
            </div>
          ))}
        </div>
      ) : null}

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
