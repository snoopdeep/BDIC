"use client";

import { useState, useEffect } from "react";
import * as api from "../../lib/api";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Button, PageHeader, Panel } from "../ui";

export default function SettingsPanel({ lang }) {
  const { data: school, loading, error, reload } = useApiData(api.getSchool);
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState("");
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    if (school) {
      setForm(school);
    }
  }, [school]);

  const handleChange = (field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setSuccess("");
    setSaveError("");
    try {
      await api.updateSchool(form);
      setSuccess(
        lang === "hi"
          ? "विद्यालय जानकारी सफलतापूर्वक अपडेट कर दी गई है।"
          : "School details updated successfully."
      );
      reload();
    } catch (err) {
      setSaveError(errorMessage(err, lang));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Panel>
        <p className="text-stone-500 animate-pulse py-8 text-center">
          {lang === "hi" ? "लोड हो रहा है..." : "Loading school settings..."}
        </p>
      </Panel>
    );
  }

  if (error) {
    return <Alert type="error">{errorMessage(error, lang)}</Alert>;
  }

  if (!form) return null;

  return (
    <div className="space-y-6">
      <PageHeader
        title={lang === "hi" ? "विद्यालय सेटिंग्स" : "School Settings"}
        description={
          lang === "hi"
            ? "विद्यालय का नाम, पता, संपर्क और अन्य जानकारी प्रबंधित करें"
            : "Manage school profile, contact information, board affiliation and branding"
        }
      />

      {success && <Alert type="success">{success}</Alert>}
      {saveError && <Alert type="error">{saveError}</Alert>}

      <form onSubmit={handleSubmit} className="space-y-6">
        <Panel title={lang === "hi" ? "मूल जानकारी" : "Basic Information"}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "विद्यालय का नाम (अंग्रेज़ी)" : "School Name (English)"}
              </label>
              <input
                type="text"
                value={form.nameEn || ""}
                onChange={(e) => handleChange("nameEn", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                required
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "विद्यालय का नाम (हिंदी)" : "School Name (Hindi)"}
              </label>
              <input
                type="text"
                value={form.nameHi || ""}
                onChange={(e) => handleChange("nameHi", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "संक्षिप्त नाम" : "Short Name"}
              </label>
              <input
                type="text"
                value={form.shortName || ""}
                onChange={(e) => handleChange("shortName", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "बोर्ड" : "Board"}
              </label>
              <input
                type="text"
                value={form.board || ""}
                onChange={(e) => handleChange("board", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "संबद्धता संख्या" : "Affiliation No."}
              </label>
              <input
                type="text"
                value={form.affiliationNo || ""}
                onChange={(e) => handleChange("affiliationNo", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "यू-डाइस कोड" : "UDISE Code"}
              </label>
              <input
                type="text"
                value={form.udiseCode || ""}
                onChange={(e) => handleChange("udiseCode", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
          </div>
        </Panel>

        <Panel title={lang === "hi" ? "संपर्क और पता" : "Contact & Address"}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "प्राथमिक फोन" : "Primary Phone"}
              </label>
              <input
                type="text"
                value={form.phonePrimary || ""}
                onChange={(e) => handleChange("phonePrimary", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "ईमेल" : "Email"}
              </label>
              <input
                type="email"
                value={form.email || ""}
                onChange={(e) => handleChange("email", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div className="md:col-span-2">
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "पता (अंग्रेज़ी)" : "Address (English)"}
              </label>
              <textarea
                value={form.addressEn || ""}
                onChange={(e) => handleChange("addressEn", e.target.value)}
                rows={2}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "प्रधानाचार्य का नाम" : "Principal Name"}
              </label>
              <input
                type="text"
                value={form.principalName || ""}
                onChange={(e) => handleChange("principalName", e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
          </div>
        </Panel>

        <div className="flex justify-end gap-3">
          <Button type="submit" disabled={saving}>
            {saving
              ? lang === "hi"
                ? "सहेजा जा रहा है..."
                : "Saving..."
              : lang === "hi"
              ? "परिवर्तन सहेजें"
              : "Save Changes"}
          </Button>
        </div>
      </form>
    </div>
  );
}
