"use client";

import { usePathname, useRouter } from "next/navigation";
import { useTransition } from "react";

import { rememberLocale } from "../lib/session";

/**
 * The Hindi/English switch.
 *
 * One tap, and the whole system changes language — including the pages the
 * reader has not opened yet, because the choice is written to a cookie that
 * proxy.js reads on the next visit.
 *
 * The switch rewrites only the first path segment, so a parent looking at
 * their child's attendance stays on that screen rather than being dropped back
 * at the home page.
 */
export default function LanguageToggle({ locale, otherLocale, label }) {
  const pathname = usePathname();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  function switchLanguage() {
    rememberLocale(otherLocale);

    const segments = pathname.split("/");
    // ["", "hi", "portal", ...] — index 1 is the language.
    segments[1] = otherLocale;
    const destination = segments.join("/") || `/${otherLocale}`;

    startTransition(() => {
      router.replace(destination);
      // The <html lang> attribute is set by the layout on the next render, and
      // the CSS keys Hindi's larger type off it.
      router.refresh();
    });
  }

  return (
    <button
      type="button"
      onClick={switchLanguage}
      disabled={isPending}
      // The label is always written in the language being offered, so a reader
      // who cannot read the current one can still find the way out.
      lang={otherLocale}
      aria-label={label}
      className="inline-flex items-center gap-1.5 rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm font-medium text-brand-800 transition hover:bg-brand-50 disabled:opacity-60"
    >
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.75"
        className="size-4"
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3c2.5 2.6 2.5 15.4 0 18M12 3c-2.5 2.6-2.5 15.4 0 18" />
      </svg>
      {label}
    </button>
  );
}
