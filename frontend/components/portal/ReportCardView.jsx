"use client";

import { useState } from "react";
import * as api from "../../lib/api";
import { formatPercent } from "../../lib/format";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Badge, Button, DataTable, EmptyState, LoadingBlock, PageHeader, Panel } from "../ui";

export default function ReportCardView({ lang }) {
  const { data: exams, loading: loadingExams } = useApiData(api.listExams);
  const { data: results, loading: loadingResults, error } = useApiData(api.listResults);
  const [selectedStudent, setSelectedStudent] = useState("");

  if (loadingExams || loadingResults) {
    return <LoadingBlock title={lang === "hi" ? "अंक-पत्र लोड हो रहा है..." : "Loading report card..."} />;
  }

  if (error) {
    return <Alert type="error">{errorMessage(error, lang)}</Alert>;
  }

  const items = Array.isArray(results) ? results : results?.items || [];

  return (
    <div className="space-y-6">
      <PageHeader
        title={lang === "hi" ? "अंक-पत्र (Report Card)" : "Student Report Card"}
        description={
          lang === "hi"
            ? "विद्यार्थी का परीक्षा परिणाम और संचयी प्रगति पत्रक"
            : "Detailed academic progress report and cumulative marksheet"
        }
        action={
          <Button onClick={() => window.print()} variant="outline">
            {lang === "hi" ? "प्रिंट लें" : "Print Report Card"}
          </Button>
        }
      />

      <Panel>
        {items.length === 0 ? (
          <EmptyState
            title={lang === "hi" ? "कोई अंक-पत्र उपलब्ध नहीं" : "No Report Cards Available"}
            description={
              lang === "hi"
                ? "परीक्षा परिणाम घोषित होने के बाद अंक-पत्र यहाँ प्रदर्शित होंगे।"
                : "Report cards will appear here once exam results are approved."
            }
          />
        ) : (
          <div className="space-y-6">
            <div className="border-b pb-4">
              <h3 className="font-bold text-lg text-stone-900">
                Bhagwan Das Inter College, Ekauna, Chandauli
              </h3>
              <p className="text-xs text-stone-500">
                {lang === "hi" ? "अकादमिक अंक-पत्र" : "Academic Progress Card"}
              </p>
            </div>

            <DataTable
              columns={[
                {
                  key: "exam",
                  header: lang === "hi" ? "परीक्षा" : "Exam",
                  render: (row) => row.examName || row.examId || "—",
                },
                {
                  key: "subject",
                  header: lang === "hi" ? "विषय" : "Subject",
                  render: (row) => row.subjectName || row.subjectCode || "—",
                },
                {
                  key: "marks",
                  header: lang === "hi" ? "प्राप्तांक / पूर्णांक" : "Marks Obtained / Max",
                  render: (row) => `${row.marksObtained || 0} / ${row.maxMarks || 100}`,
                },
                {
                  key: "grade",
                  header: lang === "hi" ? "ग्रेड" : "Grade",
                  render: (row) => <Badge variant="emerald">{row.grade || "A"}</Badge>,
                },
                {
                  key: "status",
                  header: lang === "hi" ? "स्थिति" : "Status",
                  render: (row) => (
                    <Badge variant={row.isPass !== false ? "emerald" : "amber"}>
                      {row.isPass !== false
                        ? lang === "hi" ? "उत्तीर्ण" : "Pass"
                        : lang === "hi" ? "अनुत्तीर्ण" : "Fail"}
                    </Badge>
                  ),
                },
              ]}
              data={items}
            />
          </div>
        )}
      </Panel>
    </div>
  );
}
