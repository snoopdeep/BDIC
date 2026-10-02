"use client";

import { useState } from "react";
import * as api from "../../lib/api";
import { errorMessage, useApiData } from "../../lib/portal";
import { Alert, Button, EmptyState, LoadingBlock, PageHeader, Panel, Modal } from "../ui";
import { useSession } from "../SessionContext";

export default function GalleryView({ lang, dict }) {
  const session = useSession();
  const { data, error, loading, reload } = useApiData((token, signal) =>
    api.request("/api/v1/public/site/gallery", { token, signal }),
  );
  const items = data?.items ?? data?.images ?? (Array.isArray(data) ? data : []);
  const title = dict.portal.sections.gallery;

  const [uploading, setUploading] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [caption, setCaption] = useState("");
  const [selectedFile, setSelectedFile] = useState(null);
  const [uploadError, setUploadError] = useState("");
  const [uploadSuccess, setUploadSuccess] = useState("");

  const isStaff = session?.user?.role === "super_admin" || session?.user?.role === "principal" || session?.user?.role === "office";

  const handleUpload = async (e) => {
    e.preventDefault();
    if (!selectedFile) return;
    setUploading(true);
    setUploadError("");
    setUploadSuccess("");

    try {
      const formData = new FormData();
      formData.append("file", selectedFile);
      formData.append("caption", caption);

      // Upload file via API
      const res = await fetch(`${api.API_BASE}/api/v1/files`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${session?.token}`,
        },
        body: formData,
      });

      if (!res.ok) {
        throw new Error(lang === "hi" ? "फ़ोटो अपलोड करने में त्रुटि हुई।" : "Failed to upload file.");
      }

      setUploadSuccess(
        lang === "hi"
          ? "चित्र सफलतापूर्वक गैलरी में अपलोड हो गया है!"
          : "Photo uploaded to school gallery successfully!"
      );
      setCaption("");
      setSelectedFile(null);
      setShowModal(false);
      reload();
    } catch (err) {
      setUploadError(err.message || errorMessage(err, lang));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title={title}
        action={
          isStaff && (
            <Button onClick={() => setShowModal(true)}>
              {lang === "hi" ? "+ नया फ़ोटो/वीडियो जोड़ें" : "+ Upload Photo / Video"}
            </Button>
          )
        }
      />

      {uploadSuccess && <Alert type="success">{uploadSuccess}</Alert>}

      {loading ? <Panel><LoadingBlock label={dict.common.loading} rows={5} /></Panel> : null}
      {!loading && error ? <Panel><div className="space-y-3"><Alert tone="bad">{errorMessage(error, dict, lang)}</Alert><Button variant="ghost" onClick={reload}>{dict.common.retry}</Button></div></Panel> : null}

      {!loading && !error && items.length === 0 ? <Panel><EmptyState title={dict.common.noResults} description={lang === "hi" ? "विद्यालय गैलरी में कोई फ़ोटो नहीं है। आप नया फ़ोटो अपलोड कर सकते हैं।" : "No photos in school gallery. Staff can upload new images."} /></Panel> : null}

      {!loading && !error && items.length > 0 ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
          {items.map((image) => (
            <a
              key={image.id ?? image.url}
              href={image.url ?? image.imageUrl ?? "#"}
              target="_blank"
              rel="noopener noreferrer"
              className="group overflow-hidden rounded-xl border border-stone-200 bg-white transition hover:border-emerald-500 hover:shadow-lg"
            >
              <div className="aspect-square bg-stone-100 overflow-hidden">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={image.thumbnailUrl ?? image.url ?? image.imageUrl}
                  alt={image.caption ?? image.title ?? ""}
                  className="h-full w-full object-cover transition duration-300 group-hover:scale-105"
                  loading="lazy"
                />
              </div>
              {image.caption || image.title ? (
                <p className="p-3 text-xs font-medium text-stone-700 line-clamp-2">{image.caption ?? image.title}</p>
              ) : null}
            </a>
          ))}
        </div>
      ) : null}

      {showModal && (
        <Modal
          title={lang === "hi" ? "विद्यालय फ़ोटो / वीडियो अपलोड करें" : "Upload School Photo / Video"}
          onClose={() => setShowModal(false)}
        >
          <form onSubmit={handleUpload} className="space-y-4">
            {uploadError && <Alert type="error">{uploadError}</Alert>}

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "फ़ोटो या वीडियो फ़ाइल चुनें" : "Select Photo or Video File"}
              </label>
              <input
                type="file"
                accept="image/*,video/*"
                onChange={(e) => setSelectedFile(e.target.files[0])}
                className="w-full text-sm border p-2 rounded-md"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-stone-600 mb-1">
                {lang === "hi" ? "शीर्षक / विवरण" : "Title / Caption"}
              </label>
              <input
                type="text"
                value={caption}
                onChange={(e) => setCaption(e.target.value)}
                placeholder={lang === "hi" ? "उदा. वार्षिक खेल दिवस 2026" : "e.g. Annual Sports Day 2026"}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="outline" onClick={() => setShowModal(false)}>
                {lang === "hi" ? "रद्द करें" : "Cancel"}
              </Button>
              <Button type="submit" disabled={uploading || !selectedFile}>
                {uploading
                  ? lang === "hi"
                    ? "अपलोड हो रहा है..."
                    : "Uploading..."
                  : lang === "hi"
                  ? "अपलोड करें"
                  : "Upload File"}
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}
