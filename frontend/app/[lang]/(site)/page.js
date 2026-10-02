import Link from "next/link";

import { getDictionary } from "../dictionaries";
import { fetchPublicEvents, fetchPublicNotices } from "../../../lib/api";
import { getSchool, pick } from "../../../lib/school";

/**
 * The home page.
 *
 * Statically rendered in both languages. A visitor on a slow connection gets
 * the school's name, what it is, how to reach it, and how to apply, in the
 * first HTML that arrives — no waiting for JavaScript.
 *
 * Notices and events come from the same public API used by the full site. If
 * the local API is unavailable, the rest of the page still remains useful.
 */
export default async function HomePage({ params }) {
  const { lang } = await params;
  const [dict, schoolResult, noticesResult, eventsResult] = await Promise.all([
    getDictionary(lang),
    getSchool(),
    fetchPublicNotices({ timeoutMs: 4000 }).catch(() => null),
    fetchPublicEvents({ timeoutMs: 4000 }).catch(() => null),
  ]);
  const { school } = schoolResult;
  const notices = noticesResult?.items ?? [];
  const events = eventsResult?.items ?? [];

  const schoolName = pick(school, "name", lang);
  const address = pick(school, "address", lang);

  const quickLinks = [
    { href: `/${lang}/admissions`, label: dict.nav.admissions },
    { href: `/${lang}/academics`, label: dict.nav.academics },
    { href: `/${lang}/notices`, label: dict.nav.notices },
    { href: `/${lang}/gallery`, label: dict.nav.gallery },
    { href: `/${lang}/achievements`, label: dict.nav.achievements },
    { href: `/${lang}/downloads`, label: dict.nav.downloads },
  ];

  return (
    <>
      {/* ---- Hero ---------------------------------------------------- */}
      <section className="bg-brand-800 text-white">
        <div className="mx-auto max-w-6xl px-4 py-12 sm:py-16">
          <p className="text-sm font-medium uppercase tracking-wide text-brand-200">
            {school.board}
            {school.academicYear ? (
              <>
                {" · "}
                {dict.home.academicYear} {school.academicYear}
              </>
            ) : null}
          </p>
          <h1 className="mt-3 max-w-3xl text-3xl font-bold leading-tight sm:text-4xl lg:text-5xl">
            {schoolName}
          </h1>
          <p className="mt-4 max-w-2xl text-lg text-brand-100">
            {dict.home.tagline}
          </p>
          <p className="mt-2 text-sm text-brand-200">{address}</p>

          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              href={`/${lang}/admissions/apply`}
              className="inline-flex min-h-12 items-center rounded-lg bg-accent-500 px-5 font-semibold text-white transition hover:bg-accent-600"
            >
              {dict.home.applyNow}
            </Link>
            <Link
              href={`/${lang}/notices`}
              className="inline-flex min-h-12 items-center rounded-lg border border-brand-300 px-5 font-semibold text-white transition hover:bg-brand-700"
            >
              {dict.home.viewNotices}
            </Link>
            {school.phonePrimary ? (
              <a
                href={`tel:${school.phonePrimary}`}
                className="inline-flex min-h-12 items-center rounded-lg border border-brand-300 px-5 font-semibold text-white transition hover:bg-brand-700 sm:hidden"
              >
                {dict.home.callSchool}
              </a>
            ) : null}
          </div>
        </div>
      </section>

      {/* ---- Notices and events ------------------------------------- */}
      <section className="mx-auto max-w-6xl px-4 py-10">
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="card p-5 lg:col-span-2">
            <div className="flex items-baseline justify-between gap-4">
              <h2 className="text-xl font-semibold text-brand-900">
                {dict.home.latestNotices}
              </h2>
              <Link
                href={`/${lang}/notices`}
                className="text-sm font-medium text-brand-700 underline-offset-2 hover:underline"
              >
                {dict.home.viewNotices}
              </Link>
            </div>
            {notices.length ? (
              <ul className="mt-4 divide-y divide-zinc-100">
                {notices.slice(0, 3).map((notice) => (
                  <li key={notice.id} className="py-3 first:pt-0">
                    <p className="font-medium text-zinc-900">{notice.title}</p>
                    <p className="mt-1 text-sm text-zinc-600">{notice.description}</p>
                    <p className="mt-1 text-xs text-zinc-500">{notice.createdDate}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mt-4 rounded-lg bg-brand-50 px-4 py-6 text-center text-sm text-zinc-600">
                {lang === "hi" ? "अभी कोई प्रकाशित सूचना नहीं है।" : "No published notices yet."}
              </p>
            )}
          </div>

          <div className="card p-5">
            <h2 className="text-xl font-semibold text-brand-900">
              {dict.home.upcomingEvents}
            </h2>
            {events.length ? (
              <ul className="mt-4 divide-y divide-zinc-100">
                {events.slice(0, 3).map((event) => (
                  <li key={event.id} className="py-3 first:pt-0">
                    <p className="font-medium text-zinc-900">{event.title}</p>
                    <p className="mt-1 text-sm text-zinc-600">{event.startDate}{event.startTime ? ` · ${event.startTime}` : ""}</p>
                    <p className="mt-1 text-sm text-zinc-600">{event.venueEn}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mt-4 rounded-lg bg-brand-50 px-4 py-6 text-center text-sm text-zinc-600">
                {lang === "hi" ? "अभी कोई आगामी कार्यक्रम नहीं है।" : "No upcoming events yet."}
              </p>
            )}
          </div>
        </div>
      </section>

      {/* ---- Quick links -------------------------------------------- */}
      <section className="mx-auto max-w-6xl px-4 pb-10">
        <h2 className="text-xl font-semibold text-brand-900">
          {dict.home.quickLinks}
        </h2>
        <ul className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          {quickLinks.map((link) => (
            <li key={link.href}>
              <Link
                href={link.href}
                className="card flex min-h-20 items-center justify-center p-3 text-center text-sm font-medium text-brand-800 transition hover:border-brand-300 hover:bg-brand-50"
              >
                {link.label}
              </Link>
            </li>
          ))}
        </ul>
      </section>

      {/* ---- Contact strip ------------------------------------------ */}
      <section className="mx-auto max-w-6xl px-4 pb-4">
        <div className="card grid gap-4 p-5 sm:grid-cols-3">
          <div>
            <h2 className="text-sm font-semibold text-zinc-900">
              {dict.footer.address}
            </h2>
            <p className="mt-1 text-sm text-zinc-600">{address}</p>
          </div>
          <div>
            <h2 className="text-sm font-semibold text-zinc-900">
              {dict.footer.officeHours}
            </h2>
            <p className="mt-1 text-sm text-zinc-600">
              {pick(school, "officeHours", lang)}
            </p>
          </div>
          <div>
            <h2 className="text-sm font-semibold text-zinc-900">
              {dict.footer.phone}
            </h2>
            {school.phonePrimary ? (
              <a
                href={`tel:${school.phonePrimary}`}
                className="mt-1 block text-sm text-brand-700 underline-offset-2 hover:underline"
              >
                {school.phonePrimary}
              </a>
            ) : (
              <p className="mt-1 text-sm text-zinc-400">—</p>
            )}
          </div>
        </div>
      </section>
    </>
  );
}
