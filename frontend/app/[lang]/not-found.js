import Link from "next/link";

/**
 * Not found.
 *
 * Bilingual without knowing the language: this file sits above the [lang]
 * segment's params in some cases (a notFound() thrown from the layout itself),
 * so rather than guess, it says both. Two short lines is cheaper than being
 * wrong.
 */
export default function NotFound() {
  return (
    <div id="main" className="grid min-h-dvh place-items-center bg-brand-50 px-4">
      <div className="card max-w-md p-8 text-center">
        <p className="text-5xl font-bold text-brand-300">404</p>

        <p className="mt-4 text-base font-medium text-zinc-900" lang="hi">
          यह पृष्ठ नहीं मिला।
        </p>
        <p className="mt-1 text-base text-zinc-600" lang="en">
          This page was not found.
        </p>

        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <Link
            href="/hi"
            lang="hi"
            className="inline-flex min-h-12 items-center rounded-lg bg-brand-700 px-5 font-semibold text-white hover:bg-brand-800"
          >
            मुख्य पृष्ठ
          </Link>
          <Link
            href="/en"
            lang="en"
            className="inline-flex min-h-12 items-center rounded-lg border border-brand-200 px-5 font-semibold text-brand-800 hover:bg-brand-50"
          >
            Home
          </Link>
        </div>
      </div>
    </div>
  );
}
