"use client";

import { useState } from "react";

import * as api from "../../lib/api";
import { formatDate, formatMoney, daysUntil } from "../../lib/format";
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
  StatCard,
} from "../ui";

const PAGE_SIZE = 25;

const STATUS_TONES = {
  PAID: "good",
  PARTIAL: "warn",
  UNPAID: "bad",
  OVERDUE: "bad",
  CANCELLED: "neutral",
};

const STATUS_LABELS = {
  en: {
    PAID: "Paid",
    PARTIAL: "Partial",
    UNPAID: "Unpaid",
    OVERDUE: "Overdue",
    CANCELLED: "Cancelled",
  },
  hi: {
    PAID: "भुगतान हो गया",
    PARTIAL: "आंशिक",
    UNPAID: "बकाया",
    OVERDUE: "अतिदेय",
    CANCELLED: "रद्द",
  },
};

export default function FeesList({ lang, dict }) {
  const { user } = useSession();
  const [page, setPage] = useState(0);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/fees/invoices?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const items = data?.items ?? [];
  const title = dict.portal.sections.fees;

  const isFamily = user?.role === "STUDENT" || user?.role === "PARENT";

  // Calculate summary stats
  const totalDue = items.reduce((sum, inv) => sum + (inv.outstandingPaise ?? 0), 0);
  const totalPaid = items.reduce((sum, inv) => sum + (inv.paidPaise ?? 0), 0);
  const overdueCount = items.filter(
    (inv) => inv.outstandingPaise > 0 && daysUntil(inv.dueDate) < 0,
  ).length;

  const columns = [
    {
      key: "invoiceNo",
      header: lang === "hi" ? "चालान सं." : "Invoice No.",
      isPrimary: isFamily,
    },
    ...(isFamily
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
      key: "installmentNo",
      header: lang === "hi" ? "किस्त" : "Inst.",
      hideOnMobile: true,
    },
    {
      key: "dueDate",
      header: lang === "hi" ? "अंतिम तिथि" : "Due date",
      render: (row) => {
        const days = daysUntil(row.dueDate);
        return (
          <span>
            {formatDate(row.dueDate, lang, { short: true })}
            {days !== null && days < 0 && row.outstandingPaise > 0 ? (
              <span className="ml-1 text-xs text-bad-600">
                ({Math.abs(days)}d {lang === "hi" ? "अतिदेय" : "overdue"})
              </span>
            ) : null}
          </span>
        );
      },
    },
    {
      key: "outstandingPaise",
      header: lang === "hi" ? "बकाया" : "Outstanding",
      numeric: true,
      render: (row) => (
        <span
          className={
            row.outstandingPaise > 0 ? "font-semibold text-bad-600" : ""
          }
        >
          {formatMoney(row.outstandingPaise)}
        </span>
      ),
    },
    {
      key: "status",
      header: lang === "hi" ? "स्थिति" : "Status",
      render: (row) => {
        const days = daysUntil(row.dueDate);
        const effectiveStatus =
          row.status === "UNPAID" && days !== null && days < 0
            ? "OVERDUE"
            : row.status;
        return (
          <Badge tone={STATUS_TONES[effectiveStatus] ?? "neutral"}>
            {STATUS_LABELS[lang]?.[effectiveStatus] ?? effectiveStatus}
          </Badge>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      {/* Summary cards */}
      {!loading && !error && items.length > 0 ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <StatCard
            label={lang === "hi" ? "कुल बकाया" : "Total outstanding"}
            value={formatMoney(totalDue)}
            tone={totalDue > 0 ? "bad" : "good"}
          />
          <StatCard
            label={lang === "hi" ? "कुल भुगतान" : "Total paid"}
            value={formatMoney(totalPaid)}
            tone="good"
          />
          {overdueCount > 0 ? (
            <StatCard
              label={lang === "hi" ? "अतिदेय चालान" : "Overdue invoices"}
              value={overdueCount}
              tone="bad"
            />
          ) : null}
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
