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
  RESIGNED: "neutral",
  RETIRED: "neutral",
};

export default function StaffList({ lang, dict }) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(0);
  const debouncedSearch = useDebounced(search);

  const { data, error, loading, reload } = useApiData(
    (token, signal) =>
      api.request(
        `/api/v1/people/staff?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`,
        { token, signal },
      ),
    { deps: [page] },
  );

  const allItems = data?.items ?? [];

  const items = debouncedSearch
    ? allItems.filter((staff) => {
        const term = debouncedSearch.toLowerCase();
        return (
          staff.fullNameEn?.toLowerCase().includes(term) ||
          staff.fullNameHi?.includes(debouncedSearch) ||
          staff.employeeCode?.toLowerCase().includes(term) ||
          staff.designationEn?.toLowerCase().includes(term) ||
          staff.department?.toLowerCase().includes(term)
        );
      })
    : allItems;

  const title = dict.portal.sections.staff;

  const columns = [
    {
      key: "name",
      header: lang === "hi" ? "नाम" : "Name",
      isPrimary: true,
      render: (row) => localised(row, "fullName", lang) || row.fullNameEn,
    },
    {
      key: "employeeCode",
      header: lang === "hi" ? "कर्मचारी कोड" : "Emp. Code",
    },
    {
      key: "designationEn",
      header: lang === "hi" ? "पदनाम" : "Designation",
      hideOnMobile: true,
    },
    {
      key: "department",
      header: lang === "hi" ? "विभाग" : "Department",
      hideOnMobile: true,
      render: (row) => row.department || "—",
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
                ? "नाम, कोड, या विभाग खोजें…"
                : "Search by name, code, or department…"
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
