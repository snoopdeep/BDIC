"use client";

import { useState } from "react";
import * as api from "../../lib/api";
import { formatMoney } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Button, PageHeader, Panel, StatCard } from "../ui";

export default function ReportsList({ lang }) {
  const { data: dashboard, loading, error } = useApiData(api.getDashboardStats);

  if (loading) {
    return (
      <Panel>
        <p className="text-stone-500 animate-pulse py-8 text-center">
          {lang === "hi" ? "रिपोर्ट लोड हो रही है..." : "Loading reports..."}
        </p>
      </Panel>
    );
  }

  if (error) {
    return <Alert type="error">{errorMessage(error, lang)}</Alert>;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={lang === "hi" ? "रिपोर्ट्स एवं विश्लेषण" : "Reports & Analytics"}
        description={
          lang === "hi"
            ? "शुल्क संग्रह, उपस्थिति एवं छात्र सांख्यिकी रिपोर्ट"
            : "Overview of fee collections, attendance averages, and student stats"
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label={lang === "hi" ? "कुल छात्र" : "Total Students"}
          value={dashboard?.activeStudents || 0}
        />
        <StatCard
          label={lang === "hi" ? "कुल शिक्षक" : "Total Teachers"}
          value={dashboard?.activeTeachers || 0}
        />
        <StatCard
          label={lang === "hi" ? "आज उपस्थिति" : "Today Attendance"}
          value={`${dashboard?.todayAttendancePercent || 0}%`}
        />
        <StatCard
          label={lang === "hi" ? "मासिक शुल्क संग्रह" : "Monthly Fee Collection"}
          value={formatMoney(dashboard?.monthlyFeeCollection || 0)}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Panel title={lang === "hi" ? "उपलब्ध रिपोर्ट प्रकार" : "Available Report Types"}>
          <div className="space-y-3">
            <div className="p-3 border rounded-md flex justify-between items-center hover:bg-stone-50 transition">
              <div>
                <p className="font-medium text-sm text-stone-900">
                  {lang === "hi" ? "मासिक उपस्थिति रिपोर्ट" : "Monthly Attendance Report"}
                </p>
                <p className="text-xs text-stone-500">
                  {lang === "hi" ? "कक्षा-वार उपस्थिति सारांश" : "Class-wise attendance summary"}
                </p>
              </div>
              <Button size="sm" variant="outline">
                {lang === "hi" ? "डाउनलोड" : "Download"}
              </Button>
            </div>

            <div className="p-3 border rounded-md flex justify-between items-center hover:bg-stone-50 transition">
              <div>
                <p className="font-medium text-sm text-stone-900">
                  {lang === "hi" ? "शुल्क बकाया रिपोर्ट" : "Fee Dues Report"}
                </p>
                <p className="text-xs text-stone-500">
                  {lang === "hi" ? "बकाया शुल्क सूची" : "Pending fee list by student/class"}
                </p>
              </div>
              <Button size="sm" variant="outline">
                {lang === "hi" ? "डाउनलोड" : "Download"}
              </Button>
            </div>

            <div className="p-3 border rounded-md flex justify-between items-center hover:bg-stone-50 transition">
              <div>
                <p className="font-medium text-sm text-stone-900">
                  {lang === "hi" ? "परीक्षा प्रदर्शन रिपोर्ट" : "Exam Performance Report"}
                </p>
                <p className="text-xs text-stone-500">
                  {lang === "hi" ? "विषय-वार अंक और ग्रेड विश्लेषण" : "Subject-wise marks & grade analytics"}
                </p>
              </div>
              <Button size="sm" variant="outline">
                {lang === "hi" ? "डाउनलोड" : "Download"}
              </Button>
            </div>
          </div>
        </Panel>

        <Panel title={lang === "hi" ? "शीघ्र कार्य" : "Quick Actions"}>
          <p className="text-sm text-stone-600 mb-4">
            {lang === "hi"
              ? "विशिष्ट तिथियों और कक्षाओं के लिए अनुकूलित रिपोर्ट तैयार करें।"
              : "Generate custom reports for specific date ranges and classes."}
          </p>
          <Button className="w-full">
            {lang === "hi" ? "+ नई रिपोर्ट बनाएं" : "+ Generate Custom Report"}
          </Button>
        </Panel>
      </div>
    </div>
  );
}
