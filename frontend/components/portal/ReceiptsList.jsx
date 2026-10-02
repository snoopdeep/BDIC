"use client";

import { useState } from "react";

import * as api from "../../lib/api";
import { formatDateTime, formatMoney } from "../../lib/format";
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

const MODE_LABELS = {
  en: {
    CASH: "Cash",
    CHEQUE: "Cheque",
    UPI: "UPI",
    CARD: "Card",
    NETBANKING: "Net banking",
    BANK_TRANSFER: "Bank transfer",
  },
  hi: {
    CASH: "नकद",
    CHEQUE: "चेक",
    UPI: "UPI",
    CARD: "कार्ड",
    NETBANKING: "नेट बैंकिंग",
    BANK_TRANSFER: "बैंक ट्रांसफ़र",
  },
};

export default function ReceiptsList({ lang, dict }) {
  const { user } = useSession();
  const [page, setPage] = useState(0);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/fees/receipts?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const items = data?.items ?? [];
  const title = dict.portal.sections.receipts;

  const isFamily = user?.role === "STUDENT" || user?.role === "PARENT";

  const columns = [
    {
      key: "receiptNo",
      header: lang === "hi" ? "रसीद सं." : "Receipt No.",
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
      key: "amountPaise",
      header: lang === "hi" ? "राशि" : "Amount",
      numeric: true,
      render: (row) => (
        <span className="font-semibold">{formatMoney(row.amountPaise)}</span>
      ),
    },
    {
      key: "mode",
      header: lang === "hi" ? "माध्यम" : "Mode",
      render: (row) => (
        <Badge tone="neutral">
          {MODE_LABELS[lang]?.[row.mode] ?? row.mode}
        </Badge>
      ),
    },
    {
      key: "receivedAt",
      header: lang === "hi" ? "तिथि" : "Date",
      hideOnMobile: true,
      render: (row) => formatDateTime(row.receivedAt, lang),
    },
    {
      key: "status",
      header: lang === "hi" ? "स्थिति" : "Status",
      hideOnMobile: true,
      render: (row) =>
        row.cancelledAt ? (
          <Badge tone="bad">
            {lang === "hi" ? "रद्द" : "Cancelled"}
          </Badge>
        ) : (
          <Badge tone="good">
            {lang === "hi" ? "मान्य" : "Valid"}
          </Badge>
        ),
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
