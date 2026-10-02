"use client";

import { useMemo } from "react";

import * as api from "../../lib/api";
import { formatTime } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { useSession } from "../../lib/session";
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  LoadingBlock,
  PageHeader,
  Panel,
} from "../ui";

const DAY_NAMES = {
  en: ["", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"],
  hi: ["", "सोमवार", "मंगलवार", "बुधवार", "गुरुवार", "शुक्रवार", "शनिवार"],
};

export default function TimetableView({ lang, dict }) {
  const { user } = useSession();

  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/timetable", { token, signal }),
  );

  const items = data?.items ?? [];
  const title = dict.portal.sections.timetable;

  // Group entries by day of week
  const byDay = useMemo(() => {
    const grouped = {};
    for (const entry of items) {
      const day = entry.dayOfWeek;
      if (!grouped[day]) {
        grouped[day] = [];
      }
      grouped[day].push(entry);
    }
    // Sort each day's entries by start time / period name
    for (const day of Object.keys(grouped)) {
      grouped[day].sort(
        (a, b) =>
          (a.startTime || "").localeCompare(b.startTime || "") ||
          (a.periodName || "").localeCompare(b.periodName || ""),
      );
    }
    return grouped;
  }, [items]);

  const days = Object.keys(byDay)
    .map(Number)
    .sort((a, b) => a - b);

  const isTeacher = user?.role === "TEACHER";

  return (
    <div className="space-y-6">
      <PageHeader title={title} />

      {loading ? (
        <Panel>
          <LoadingBlock label={dict.common.loading} rows={5} />
        </Panel>
      ) : null}

      {!loading && error ? (
        <Panel>
          <div className="space-y-3">
            <Alert tone="bad">{errorMessage(error, dict, lang)}</Alert>
            <Button variant="ghost" onClick={reload}>
              {dict.common.retry}
            </Button>
          </div>
        </Panel>
      ) : null}

      {!loading && !error && days.length === 0 ? (
        <Panel>
          <EmptyState title={dict.common.noResults} />
        </Panel>
      ) : null}

      {!loading && !error && days.length > 0
        ? days.map((day) => (
            <Panel
              key={day}
              title={DAY_NAMES[lang]?.[day] ?? `Day ${day}`}
            >
              <div className="divide-y divide-zinc-100">
                {byDay[day].map((entry) => (
                  <div
                    key={entry.id}
                    className="flex flex-wrap items-center gap-x-4 gap-y-1 py-3 first:pt-0 last:pb-0"
                  >
                    {/* Time */}
                    <div className="w-28 shrink-0">
                      <p className="text-sm font-medium tabular text-brand-900">
                        {formatTime(entry.startTime, lang)}
                        {entry.endTime
                          ? ` – ${formatTime(entry.endTime, lang)}`
                          : ""}
                      </p>
                      <p className="text-xs text-zinc-500">
                        {entry.periodName}
                      </p>
                    </div>

                    {/* Subject */}
                    <div className="min-w-0 flex-1">
                      <p className="font-medium text-brand-900">
                        {entry.subjectName}
                      </p>
                      {!isTeacher ? (
                        <p className="text-sm text-zinc-600">
                          {entry.substituteTeacherName ? (
                            <>
                              <span className="line-through text-zinc-400">
                                {entry.teacherName}
                              </span>{" "}
                              <Badge tone="warn">
                                {entry.substituteTeacherName}
                              </Badge>
                            </>
                          ) : (
                            entry.teacherName
                          )}
                        </p>
                      ) : (
                        <p className="text-sm text-zinc-600">
                          {entry.className}
                          {entry.sectionName
                            ? ` – ${entry.sectionName}`
                            : ""}
                        </p>
                      )}
                    </div>

                    {/* Room */}
                    {entry.room ? (
                      <span className="text-xs text-zinc-500">
                        {entry.room}
                      </span>
                    ) : null}
                  </div>
                ))}
              </div>
            </Panel>
          ))
        : null}
    </div>
  );
}
