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

const CLASS_OPTIONS = [
  { value: "", labelEn: "All Classes (LKG - 12th)", labelHi: "सभी कक्षाएं (LKG - 12th)" },
  { value: "LKG", labelEn: "LKG", labelHi: "एल.के.जी." },
  { value: "UKG", labelEn: "UKG", labelHi: "यू.के.जी." },
  { value: "Nursery", labelEn: "Nursery", labelHi: "नर्सरी" },
  { value: "Class 1", labelEn: "Class 1", labelHi: "कक्षा 1" },
  { value: "Class 2", labelEn: "Class 2", labelHi: "कक्षा 2" },
  { value: "Class 3", labelEn: "Class 3", labelHi: "कक्षा 3" },
  { value: "Class 4", labelEn: "Class 4", labelHi: "कक्षा 4" },
  { value: "Class 5", labelEn: "Class 5", labelHi: "कक्षा 5" },
  { value: "Class 6", labelEn: "Class 6", labelHi: "कक्षा 6" },
  { value: "Class 7", labelEn: "Class 7", labelHi: "कक्षा 7" },
  { value: "Class 8", labelEn: "Class 8", labelHi: "कक्षा 8" },
  { value: "Class 9", labelEn: "Class 9", labelHi: "कक्षा 9" },
  { value: "Class 10", labelEn: "Class 10", labelHi: "कक्षा 10" },
  { value: "Class 11", labelEn: "Class 11", labelHi: "कक्षा 11" },
  { value: "Class 12", labelEn: "Class 12", labelHi: "कक्षा 12" },
];

export default function StudentsList({ lang, dict }) {
  const [search, setSearch] = useState("");
  const [selectedClass, setSelectedClass] = useState("");
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

  // Filter by search term and selected class
  const items = allItems.filter((student) => {
    const term = debouncedSearch.toLowerCase();
    const matchesSearch = !debouncedSearch || (
      student.fullNameEn?.toLowerCase().includes(term) ||
      student.fullNameHi?.includes(debouncedSearch) ||
      student.admissionNo?.toLowerCase().includes(term) ||
      student.className?.toLowerCase().includes(term) ||
      student.sectionName?.toLowerCase().includes(term)
    );

    const matchesClass = !selectedClass || (
      student.className?.toLowerCase().includes(selectedClass.toLowerCase()) ||
      student.classCode?.toLowerCase() === selectedClass.toLowerCase()
    );

    return matchesSearch && matchesClass;
  });

  const title = dict.portal.sections.students;

  const columns = [
    {
      key: "admissionNo",
      header: dict.portal.fields?.admissionNo ?? "Adm No",
      render: (row) => (
        <span className="font-mono text-xs font-semibold text-zinc-900">
          {row.admissionNo}
        </span>
      ),
    },
    {
      key: "name",
      header: dict.portal.fields?.name ?? "Name",
      render: (row) => (
        <div>
          <p className="font-medium text-zinc-900">
            {localised(row, "fullName", lang)}
          </p>
          {row.fatherNameEn ? (
            <p className="text-xs text-zinc-600">
              S/D of {localised(row, "fatherName", lang)}
            </p>
          ) : null}
        </div>
      ),
    },
    {
      key: "class",
      header: dict.portal.fields?.class ?? "Class & Sec",
      render: (row) => (
        <span className="inline-flex items-center gap-1 rounded bg-stone-100 px-2 py-1 text-xs font-medium text-stone-800">
          {row.className || row.classCode} {row.sectionName ? `- ${row.sectionName}` : ""}
        </span>
      ),
    },
    {
      key: "status",
      header: dict.portal.fields?.status ?? "Status",
      render: (row) => (
        <Badge tone={STATUS_TONES[row.status] ?? "neutral"}>
          {dict.portal.status?.[row.status] ?? row.status}
        </Badge>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title={title}
        description={
          lang === "hi"
            ? "कक्षा एल.के.जी. से 12वीं तक के विद्यार्थियों की सूची"
            : "Student directory for Classes LKG to 12th"
        }
      />

      <Panel>
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="w-full sm:max-w-xs">
            <SearchInput
              value={search}
              onChange={setSearch}
              placeholder={dict.common.searchPlaceholder}
            />
          </div>

          <div className="flex items-center gap-2">
            <label className="text-xs font-semibold text-stone-600">
              {lang === "hi" ? "कक्षा चुनें:" : "Filter Class:"}
            </label>
            <select
              value={selectedClass}
              onChange={(e) => setSelectedClass(e.target.value)}
              className="px-3 py-2 border rounded-md text-xs font-medium bg-white text-stone-800 shadow-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
            >
              {CLASS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {lang === "hi" ? opt.labelHi : opt.labelEn}
                </option>
              ))}
            </select>
          </div>
        </div>

        {loading ? (
          <LoadingBlock label={dict.common.loading} rows={6} />
        ) : null}

        {!loading && error ? (
          <div className="space-y-3">
            <Alert tone="bad">{errorMessage(error, dict, lang)}</Alert>
            <Button variant="ghost" onClick={reload}>
              {dict.common.retry}
            </Button>
          </div>
        ) : null}

        {!loading && !error && items.length === 0 ? (
          <EmptyState title={dict.common.noResults} />
        ) : null}

        {!loading && !error && items.length > 0 ? (
          <>
            <DataTable columns={columns} data={items} />

            {allItems.length >= PAGE_SIZE ? (
              <div className="mt-4 flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                >
                  {dict.common.previous || "Previous"}
                </Button>
                <span className="text-xs font-medium text-stone-600">
                  {dict.common.page || "Page"} {page + 1}
                </span>
                <Button
                  variant="outline"
                  disabled={items.length < PAGE_SIZE}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {dict.common.next || "Next"}
                </Button>
              </div>
            ) : null}
          </>
        ) : null}
      </Panel>
    </div>
  );
}
