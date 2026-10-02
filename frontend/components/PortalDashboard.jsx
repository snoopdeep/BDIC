"use client";

import { pick } from "../lib/school";
import { useSession } from "../lib/session";

/**
 * The portal's home screen.
 *
 * In this stage of the build it confirms who is signed in, in which language,
 * against which academic session — which is exactly what needs verifying
 * before the modules that sit on top of it are worth writing.
 *
 * The role-specific panels (today's classes for a teacher, each child's
 * attendance for a parent, the collection figure for accounts) arrive with
 * their own stages of the build.
 */
export default function PortalDashboard({ lang, dict }) {
  const { user, school, currentSession } = useSession();

  // pick(), not a hardcoded Hindi-first fallback: an English reader was
  // otherwise always shown the Hindi spelling of their own name.
  const name = pick(user, "fullName", lang);
  const greeting = greetingFor(dict);
  const schoolName =
    (lang === "hi" ? school?.nameHi : school?.nameEn) || school?.nameEn || "";

  const facts = [
    { label: dict.portal.signedInAs, value: name },
    { label: dict.portal.role, value: dict.roles[user.role] ?? user.role },
    { label: dict.portal.language, value: dict.meta.localeName },
    currentSession?.name
      ? { label: dict.home.academicYear, value: currentSession.name }
      : null,
    school?.board ? { label: dict.home.board, value: school.board } : null,
  ].filter(Boolean);

  return (
    <div className="space-y-6">
      <section className="card p-5 sm:p-6">
        <p className="text-sm text-zinc-500">{greeting}</p>
        <h1 className="mt-1 text-2xl font-bold text-brand-900">{name}</h1>
        {schoolName ? (
          <p className="mt-1 text-sm text-zinc-600">{schoolName}</p>
        ) : null}
      </section>

      <section className="card p-5 sm:p-6">
        <h2 className="text-lg font-semibold text-brand-900">
          {dict.portal.myProfile}
        </h2>
        <dl className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {facts.map((fact) => (
            <div key={fact.label}>
              <dt className="text-xs uppercase tracking-wide text-zinc-500">
                {fact.label}
              </dt>
              <dd className="mt-0.5 text-sm font-medium text-zinc-900">
                {fact.value}
              </dd>
            </div>
          ))}
        </dl>
      </section>

    </div>
  );
}

/**
 * A greeting that matches the time of day where the school is.
 *
 * Asia/Kolkata explicitly, not the device's time zone: the greeting should
 * match the school's morning, and a phone left on the wrong time zone should
 * not produce "good evening" during assembly.
 */
function greetingFor(dict) {
  const hour = Number(
    new Intl.DateTimeFormat("en-GB", {
      hour: "numeric",
      hour12: false,
      timeZone: "Asia/Kolkata",
    }).format(new Date()),
  );

  if (hour < 12) {
    return dict.portal.goodMorning;
  }
  if (hour < 17) {
    return dict.portal.goodAfternoon;
  }
  return dict.portal.goodEvening;
}
