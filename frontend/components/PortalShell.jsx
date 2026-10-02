"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { pick } from "../lib/school";
import { SessionProvider, rememberLocale, useSession } from "../lib/session";

/**
 * Chrome and the front door for the portal.
 *
 * The navigation a user sees is built from their role, so a teacher is never
 * shown a link to the fee ledger. That is a courtesy, not a security boundary:
 * the API checks the role on every request and scopes every query, so typing
 * the URL by hand gets a 403, not the data.
 */

/** Which sections each role sees, in the order they matter to that role. */
const NAV_BY_ROLE = {
  SUPER_ADMIN: [
    "dashboard", "admissions", "students", "staff", "attendance", "timetable",
    "exams", "fees", "notices", "gallery", "reports", "settings", "auditLog",
    "outbox",
  ],
  PRINCIPAL: [
    "dashboard", "admissions", "students", "staff", "attendance", "timetable",
    "exams", "fees", "notices", "gallery", "reports", "settings", "auditLog",
  ],
  OFFICE: [
    "dashboard", "admissions", "students", "attendance", "certificates",
    "notices", "gallery", "outbox",
  ],
  ACCOUNTS: ["dashboard", "fees", "receipts", "reports"],
  TEACHER: [
    "dashboard", "attendance", "timetable", "homework", "studyMaterial",
    "exams", "messages",
  ],
  STUDENT: [
    "dashboard", "timetable", "attendance", "homework", "studyMaterial",
    "exams", "results", "reportCard", "fees", "notices",
  ],
  PARENT: [
    "dashboard", "attendance", "homework", "results", "reportCard", "fees",
    "receipts", "notices", "messages",
  ],
};

/** Where each section lives. "dashboard" is the portal root. */
const SECTION_PATHS = {
  dashboard: "",
  admissions: "/admissions",
  students: "/students",
  staff: "/staff",
  attendance: "/attendance",
  timetable: "/timetable",
  homework: "/homework",
  studyMaterial: "/study-material",
  exams: "/exams",
  results: "/results",
  reportCard: "/report-card",
  fees: "/fees",
  receipts: "/receipts",
  certificates: "/certificates",
  notices: "/notices",
  messages: "/messages",
  gallery: "/gallery",
  reports: "/reports",
  settings: "/settings",
  auditLog: "/audit-log",
  outbox: "/outbox",
};

export default function PortalShell({ lang, dict, children }) {
  return (
    <SessionProvider>
      <PortalGuard lang={lang} dict={dict}>
        {children}
      </PortalGuard>
    </SessionProvider>
  );
}

function PortalGuard({ lang, dict, children }) {
  const { status, user } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (status === "signedOut") {
      router.replace(`/${lang}/login`);
    }
  }, [status, router, lang]);

  if (status === "loading") {
    return (
      <div className="grid min-h-dvh place-items-center bg-brand-50 px-4">
        <p className="text-sm text-zinc-600">{dict.common.loading}</p>
      </div>
    );
  }

  if (status === "signedOut" || !user) {
    // The redirect above is already on its way; this is the one frame in
    // between, and it says something honest rather than flashing the portal.
    return (
      <div className="grid min-h-dvh place-items-center bg-brand-50 px-4">
        <p className="text-sm text-zinc-600">{dict.portal.sessionExpired}</p>
      </div>
    );
  }

  return (
    <PortalFrame lang={lang} dict={dict}>
      {children}
    </PortalFrame>
  );
}

function PortalFrame({ lang, dict, children }) {
  const { user, school, currentSession, signOut, token } = useSession();
  const pathname = usePathname();
  const router = useRouter();
  const [navOpen, setNavOpen] = useState(false);

  useEffect(() => {
    setNavOpen(false);
  }, [pathname]);

  const roleKey = user.role?.toUpperCase() || user.role;
  const sections = NAV_BY_ROLE[roleKey] ?? NAV_BY_ROLE[user.role] ?? ["dashboard"];
  const portalRoot = `/${lang}/portal`;

  async function onSignOut() {
    await signOut(token);
    router.replace(`/${lang}/login`);
  }

  function switchLanguage() {
    const other = dict.meta.otherLocale;
    rememberLocale(other);
    const segments = pathname.split("/");
    segments[1] = other;
    router.replace(segments.join("/"));
    router.refresh();
  }

  const navLinks = sections.map((section) => ({
    section,
    href: `${portalRoot}${SECTION_PATHS[section] ?? ""}`,
    label: dict.portal.sections[section] ?? dict.portal.dashboard,
  }));
  // The dashboard's label comes from a different key than the rest.
  navLinks[0] = {
    section: "dashboard",
    href: portalRoot,
    label: dict.portal.dashboard,
  };

  function isActive(href) {
    if (href === portalRoot) {
      return pathname === portalRoot || pathname === `${portalRoot}/`;
    }
    // The trailing slash matters. A bare startsWith would light up
    // /portal/report-card while the reader is on /portal/reports.
    return pathname === href || pathname.startsWith(`${href}/`);
  }

  const schoolName =
    (lang === "hi" ? school?.nameHi : school?.nameEn) ||
    school?.nameEn ||
    dict.portal.title;

  return (
    <div className="min-h-dvh bg-brand-50">
      {/* ---- Top bar ------------------------------------------------ */}
      <header className="sticky top-0 z-40 border-b border-brand-100 bg-white">
        <div className="flex items-center gap-2 px-3 py-2 sm:px-4">
          <button
            type="button"
            onClick={() => setNavOpen(true)}
            aria-label={dict.nav.openMenu}
            className="grid size-11 place-items-center rounded-lg border border-brand-200 text-brand-800 lg:hidden"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="size-5"
            >
              <path d="M4 7h16M4 12h16M4 17h16" strokeLinecap="round" />
            </svg>
          </button>

          <Link href={portalRoot} className="flex min-w-0 items-center gap-2">
            <span
              aria-hidden="true"
              className="grid size-9 shrink-0 place-items-center rounded-lg bg-brand-700 text-xs font-bold text-white"
            >
              BD
            </span>
            <span className="min-w-0">
              <span className="block truncate text-sm font-semibold text-brand-900">
                {schoolName}
              </span>
              <span className="block truncate text-xs text-zinc-500">
                {dict.portal.title}
                {currentSession?.name ? ` · ${currentSession.name}` : ""}
              </span>
            </span>
          </Link>

          <div className="ml-auto flex items-center gap-2">
            <button
              type="button"
              onClick={switchLanguage}
              lang={dict.meta.otherLocale}
              className="rounded-lg border border-brand-200 px-3 py-2 text-sm font-medium text-brand-800 hover:bg-brand-50"
            >
              {dict.meta.otherLocaleName}
            </button>
            <button
              type="button"
              onClick={onSignOut}
              className="rounded-lg border border-brand-200 px-3 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50"
            >
              {dict.portal.signOut}
            </button>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-7xl">
        {/* ---- Sidebar, desktop ------------------------------------ */}
        <aside className="hidden w-60 shrink-0 border-e border-brand-100 bg-white lg:block">
          <UserBadge user={user} dict={dict} lang={lang} />
          <NavList links={navLinks} isActive={isActive} />
        </aside>

        {/* ---- Sidebar, phone -------------------------------------- */}
        {navOpen ? (
          <div className="fixed inset-0 z-50 lg:hidden">
            <button
              type="button"
              aria-label={dict.nav.closeMenu}
              onClick={() => setNavOpen(false)}
              className="absolute inset-0 bg-zinc-900/50"
            />
            <div
              role="dialog"
              aria-modal="true"
              aria-label={dict.nav.menu}
              className="absolute inset-y-0 start-0 flex w-[80%] max-w-xs flex-col bg-white shadow-xl"
            >
              <UserBadge user={user} dict={dict} lang={lang} />
              <div className="flex-1 overflow-y-auto">
                <NavList links={navLinks} isActive={isActive} />
              </div>
            </div>
          </div>
        ) : null}

        <main id="main" className="min-w-0 flex-1 px-4 py-6">
          {user.mustReset ? (
            <p className="mb-4 rounded-lg bg-warn-100 px-4 py-3 text-sm text-warn-700">
              {dict.login.mustReset}
            </p>
          ) : null}
          {children}
        </main>
      </div>
    </div>
  );
}

function UserBadge({ user, dict, lang }) {
  const name = pick(user, "fullName", lang);
  return (
    <div className="border-b border-brand-100 p-4">
      <p className="text-xs uppercase tracking-wide text-zinc-500">
        {dict.portal.signedInAs}
      </p>
      <p className="mt-1 truncate text-sm font-semibold text-brand-900">
        {name}
      </p>
      <p className="mt-0.5 text-xs text-zinc-600">
        {dict.roles[user.role] ?? user.role}
      </p>
    </div>
  );
}

function NavList({ links, isActive }) {
  return (
    <nav className="p-2">
      {links.map((link) => {
        const active = isActive(link.href);
        return (
          <Link
            key={link.href}
            href={link.href}
            aria-current={active ? "page" : undefined}
            className={
              active
                ? "flex min-h-12 items-center rounded-lg bg-brand-100 px-3 text-sm font-semibold text-brand-900"
                : "flex min-h-12 items-center rounded-lg px-3 text-sm font-medium text-zinc-700 hover:bg-brand-50"
            }
          >
            {link.label}
          </Link>
        );
      })}
    </nav>
  );
}
