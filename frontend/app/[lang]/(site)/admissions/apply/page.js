"use client";

import { useState } from "react";
import Link from "next/link";
import * as api from "../../../../../lib/api";

const CLASS_LIST = [
  { id: "LKG", name: "LKG" },
  { id: "UKG", name: "UKG" },
  { id: "NUR", name: "Nursery" },
  { id: "I", name: "Class 1" },
  { id: "II", name: "Class 2" },
  { id: "III", name: "Class 3" },
  { id: "IV", name: "Class 4" },
  { id: "V", name: "Class 5" },
  { id: "VI", name: "Class 6" },
  { id: "VII", name: "Class 7" },
  { id: "VIII", name: "Class 8" },
  { id: "IX", name: "Class 9" },
  { id: "X", name: "Class 10" },
  { id: "XI", name: "Class 11 (Intermediate)" },
  { id: "XII", name: "Class 12 (Intermediate)" },
];

export default function ApplyPage({ params }) {
  const [form, setForm] = useState({
    applicantName: "",
    dateOfBirth: "",
    gender: "MALE",
    category: "GENERAL",
    classAppliedId: "IX",
    fatherName: "",
    motherName: "",
    guardianPhone: "",
    addressLine: "",
    village: "Ekauna",
    district: "Chandauli",
    state: "Uttar Pradesh",
    previousSchool: "",
  });

  const [submitting, setSubmitting] = useState(false);
  const [successResult, setSuccessResult] = useState(null);
  const [errorMsg, setErrorMsg] = useState("");

  const handleChange = (field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setErrorMsg("");

    try {
      const res = await fetch(`${api.API_BASE}/api/v1/public/admissions/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.messageEn || data.messageHi || "Failed to submit application");
      }

      setSuccessResult(data);
    } catch (err) {
      setErrorMsg(err.message || "Could not submit application.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mx-auto max-w-3xl px-4 py-12">
      <div className="border-b pb-4 mb-6">
        <h1 className="text-3xl font-extrabold text-stone-900">
          Online Admission Application 2026-27
        </h1>
        <p className="text-sm text-stone-600 mt-1">
          Bhagwan Das Inter College, Ekauna, Chandauli • Classes LKG to 12th
        </p>
      </div>

      {successResult ? (
        <div className="card p-8 bg-emerald-50 border border-emerald-200 rounded-2xl space-y-4">
          <div className="inline-flex items-center gap-2 text-emerald-800 font-bold text-xl">
            <span>✅</span> Application Submitted Successfully!
          </div>
          <p className="text-sm text-stone-700">
            Thank you for applying. Please note your application details below for status tracking:
          </p>
          <div className="p-4 bg-white border border-emerald-200 rounded-xl space-y-2 text-sm font-mono">
            <p><strong className="font-sans">Application Number:</strong> {successResult.applicationNo || successResult.id || "APP/2026/001"}</p>
            {successResult.lookupToken && (
              <p><strong className="font-sans">Status Lookup Token:</strong> {successResult.lookupToken}</p>
            )}
          </div>
          <p className="text-xs text-stone-500">
            Please present this Application Number at the school office along with your TC, Aadhaar card, and marksheets for verification.
          </p>
          <div className="pt-4">
            <Link
              href="/"
              className="inline-flex items-center px-4 py-2 bg-emerald-700 text-white rounded-xl text-sm font-semibold hover:bg-emerald-600"
            >
              Return to Homepage →
            </Link>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-6 bg-white p-6 border rounded-2xl shadow-sm">
          {errorMsg && (
            <div className="p-4 bg-amber-50 text-amber-900 border border-amber-200 rounded-xl text-sm font-medium">
              {errorMsg}
            </div>
          )}

          {/* Section 1: Applicant Info */}
          <div>
            <h3 className="text-lg font-bold text-stone-900 border-b pb-2 mb-4">
              1. Student Basic Details / छात्र का विवरण
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Full Name of Student / छात्र का नाम *
                </label>
                <input
                  type="text"
                  required
                  value={form.applicantName}
                  onChange={(e) => handleChange("applicantName", e.target.value)}
                  placeholder="e.g. Rahul Kumar"
                  className="w-full px-3 py-2 border rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Class Applying For / कक्षा चुनें *
                </label>
                <select
                  value={form.classAppliedId}
                  onChange={(e) => handleChange("classAppliedId", e.target.value)}
                  className="w-full px-3 py-2 border rounded-xl text-sm bg-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  {CLASS_LIST.map((cls) => (
                    <option key={cls.id} value={cls.id}>{cls.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Date of Birth / जन्म तिथि
                </label>
                <input
                  type="date"
                  value={form.dateOfBirth}
                  onChange={(e) => handleChange("dateOfBirth", e.target.value)}
                  className="w-full px-3 py-2 border rounded-xl text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Gender / लिंग
                </label>
                <select
                  value={form.gender}
                  onChange={(e) => handleChange("gender", e.target.value)}
                  className="w-full px-3 py-2 border rounded-xl text-sm bg-white"
                >
                  <option value="MALE">Male / बालक</option>
                  <option value="FEMALE">Female / बालिका</option>
                  <option value="OTHER">Other</option>
                </select>
              </div>
            </div>
          </div>

          {/* Section 2: Guardian Info */}
          <div>
            <h3 className="text-lg font-bold text-stone-900 border-b pb-2 mb-4">
              2. Parent & Guardian Details / अभिभावक विवरण
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Father's Name / पिता का नाम *
                </label>
                <input
                  type="text"
                  required
                  value={form.fatherName}
                  onChange={(e) => handleChange("fatherName", e.target.value)}
                  placeholder="e.g. Ramesh Singh"
                  className="w-full px-3 py-2 border rounded-xl text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Mother's Name / माता का नाम
                </label>
                <input
                  type="text"
                  value={form.motherName}
                  onChange={(e) => handleChange("motherName", e.target.value)}
                  className="w-full px-3 py-2 border rounded-xl text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Mobile Number / मोबाइल नंबर *
                </label>
                <input
                  type="tel"
                  required
                  value={form.guardianPhone}
                  onChange={(e) => handleChange("guardianPhone", e.target.value)}
                  placeholder="e.g. 9876543210"
                  className="w-full px-3 py-2 border rounded-xl text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-stone-700 mb-1">
                  Village / Town / ग्राम या नगर
                </label>
                <input
                  type="text"
                  value={form.village}
                  onChange={(e) => handleChange("village", e.target.value)}
                  className="w-full px-3 py-2 border rounded-xl text-sm"
                />
              </div>
            </div>
          </div>

          <div className="pt-4 flex justify-end gap-3 border-t">
            <Link
              href="/"
              className="px-5 py-2.5 border rounded-xl text-sm font-semibold text-stone-600 hover:bg-stone-50"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={submitting}
              className="px-6 py-2.5 bg-emerald-600 text-white rounded-xl text-sm font-semibold hover:bg-emerald-500 shadow-md disabled:opacity-50"
            >
              {submitting ? "Submitting Application..." : "Submit Application →"}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
