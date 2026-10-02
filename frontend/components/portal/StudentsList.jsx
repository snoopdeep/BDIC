"use client";

import { useState } from "react";

import * as api from "../../lib/api";
import { localised } from "../../lib/format";
import { errorMessage, useApiData, useDebounced } from "../../lib/portal";
import {
  Alert,
  Badge,
  Button,
  DataTable,
  EmptyState,
  LoadingBlock,
  PageHeader,
  Panel,
  SearchInput,
} from "../ui";

const PAGE_SIZE = 25;

const STATUS_TONES = {
  ACTIVE: "good",
  INACTIVE: "warn",
  TRANSFERRED: "neutral",
  WITHDRAWN: "bad",
  ALUMNUS: "neutral",
};

export default function StudentsList({ lang, dict }) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const debouncedSearch = useDebounced(search);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/people/students?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const allItems = data?.items ?? [];

  // Client-side search filtering since the API doesn't support a search param
  const items = debouncedSearch
    ? allItems.filter((student) => {
        const term = debouncedSearch.toLowerCase();
        return (
          student.fullNameEn?.toLowerCase().includes(term) ||
          student.fullNameHi?.includes(debouncedSearch) ||
          student.admissionNo?.toLowerCase().includes(term) ||
          student.className?.toLowerCase().includes(term) ||
          student.sectionName?.toLowerCase().includes(term)
        );
      })
    : allItems;

  const title = dict.portal.sections.students;

  const columns = [
    {
      key: "name",
      header: lang === "hi" ? "नाम" : "Name",
      isPrimary: true,
      render: (row) => localised(row, "fullName", lang) || row.fullNameEn,
    },
    {
      key: "admissionNo",
      header: lang === "hi" ? "अनुक्रमांक" : "Adm. No.",
    },
    {
      key: "className",
      header: lang === "hi" ? "कक्षा" : "Class",
      render: (row) =>
        row.className
          ? `${row.className}${row.sectionName ? ` – ${row.sectionName}` : ""}`
          : "—",
    },
    {
      key: "status",
      header: lang === "hi" ? "स्थिति" : "Status",
      hideOnMobile: true,
      render: (row) => (
        <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>
          {row.status}
        </Badge>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title={title}
        subtitle={
          data
            ? `${lang === "hi" ? "कुल" : "Total"}: ${allItems.length}`
            : null
        }
      />

      <Panel>
        <div className="mb-4">
          <SearchInput
            value={search}
            onChange={(val) => {
              setSearch(val);
              setPage(0);
            }}
            placeholder={
              lang === "hi"
                ? "नाम, अनुक्रमांक, या कक्षा खोजें…"
                : "Search by name, admission no., or class…"
            }
          />
        </div>

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
              empty={
                <EmptyState
                  title={
                    debouncedSearch
                      ? lang === "hi"
                        ? "कोई मिलान नहीं मिला।"
                        : "No matches found."
                      : dict.common.noResults
                  }
                />
              }
            />
            {!debouncedSearch && allItems.length >= PAGE_SIZE ? (
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
                  disabled={allItems.length < PAGE_SIZE}
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
