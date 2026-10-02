import Link from "next/link";

import { getDictionary } from "../dictionaries";
import { fetchPublicEvents, fetchPublicNotices } from "../../../lib/api";
import { getSchool, pick } from "../../../lib/school";

const SCHOOL_IMAGES = {
  hero: "https://images.unsplash.com/photo-1562774053-701939374585?auto=format&fit=crop&w=1200&q=80", // Modern campus
  principal: "https://images.unsplash.com/photo-1560250097-0b93528c311a?auto=format&fit=crop&w=400&q=80",
  labs: "https://images.unsplash.com/photo-1532094349884-543bc11b234d?auto=format&fit=crop&w=600&q=80",
  library: "https://images.unsplash.com/photo-1521587760476-6c12a4b040da?auto=format&fit=crop&w=600&q=80",
  sports: "https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=600&q=80",
  classroom: "https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=600&q=80",
};

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
    { href: `/${lang}/admissions`, label: dict.nav.admissions, icon: "📋" },
    { href: `/${lang}/academics`, label: dict.nav.academics, icon: "🎓" },
    { href: `/${lang}/notices`, label: dict.nav.notices, icon: "📢" },
    { href: `/${lang}/gallery`, label: dict.nav.gallery, icon: "🖼️" },
    { href: `/${lang}/achievements`, label: dict.nav.achievements, icon: "🏆" },
    { href: `/${lang}/downloads`, label: dict.nav.downloads, icon: "📂" },
  ];

  return (
    <>
      {/* ---- Hero Section with Campus Background Banner ---- */}
      <section className="relative overflow-hidden bg-stone-900 text-white">
        <div className="absolute inset-0 z-0 opacity-25">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={SCHOOL_IMAGES.hero}
            alt="School Campus"
            className="h-full w-full object-cover"
          />
        </div>
        <div className="relative z-10 mx-auto max-w-6xl px-4 py-16 sm:py-24">
          <div className="inline-flex items-center gap-2 rounded-full bg-emerald-500/20 px-4 py-1.5 text-xs font-semibold text-emerald-300 border border-emerald-500/30">
            <span>🏛️ {school.board || "UP Board (UPMSP)"}</span>
            <span>•</span>
            <span>Classes LKG to 12th</span>
          </div>

          <h1 className="mt-4 max-w-3xl text-3xl font-extrabold leading-tight sm:text-5xl lg:text-6xl text-stone-100">
            {schoolName}
          </h1>
          <p className="mt-4 max-w-2xl text-lg text-stone-300 leading-relaxed">
            {dict.home.tagline || "Empowering minds from LKG to 12th grade with quality education, values, and excellence."}
          </p>
          <p className="mt-2 text-sm text-emerald-400 font-medium">📍 {address}</p>

          <div className="mt-8 flex flex-wrap gap-4">
            <Link
              href={`/${lang}/admissions`}
              className="inline-flex min-h-12 items-center rounded-xl bg-emerald-600 px-6 font-semibold text-white transition hover:bg-emerald-500 shadow-lg shadow-emerald-950/50"
            >
              {dict.home.applyNow || "Online Admissions Open →"}
            </Link>
            <Link
              href={`/${lang}/notices`}
              className="inline-flex min-h-12 items-center rounded-xl border border-stone-600 bg-stone-800/80 px-6 font-semibold text-stone-200 transition hover:bg-stone-700 hover:text-white"
            >
              {dict.home.viewNotices || "School Notices"}
            </Link>
          </div>
        </div>
      </section>

      {/* ---- Key Highlights & Stats Bar ----------------------------- */}
      <section className="bg-emerald-800 text-white border-y border-emerald-700">
        <div className="mx-auto max-w-6xl px-4 py-6 grid grid-cols-2 md:grid-cols-4 gap-6 text-center">
          <div>
            <p className="text-2xl font-extrabold text-emerald-200">LKG – 12th</p>
            <p className="text-xs font-medium text-emerald-100 mt-1">Complete Education</p>
          </div>
          <div>
            <p className="text-2xl font-extrabold text-emerald-200">UP Board</p>
            <p className="text-xs font-medium text-emerald-100 mt-1">Affiliated Curriculum</p>
          </div>
          <div>
            <p className="text-2xl font-extrabold text-emerald-200">Sci / Com / Arts</p>
            <p className="text-xs font-medium text-emerald-100 mt-1">Stream Choices</p>
          </div>
          <div>
            <p className="text-2xl font-extrabold text-emerald-200">Bilingual</p>
            <p className="text-xs font-medium text-emerald-100 mt-1">Hindi & English Medium</p>
          </div>
        </div>
      </section>

      {/* ---- Notices and Events ------------------------------------- */}
      <section className="mx-auto max-w-6xl px-4 py-12">
        <div className="grid gap-8 lg:grid-cols-3">
          <div className="card p-6 lg:col-span-2 bg-white border border-stone-200 rounded-2xl shadow-sm">
            <div className="flex items-baseline justify-between gap-4 border-b pb-4">
              <h2 className="text-xl font-bold text-stone-900 flex items-center gap-2">
                <span>📢</span> {dict.home.latestNotices || "Latest Notices"}
              </h2>
              <Link
                href={`/${lang}/notices`}
                className="text-xs font-semibold text-emerald-700 hover:underline"
              >
                {dict.home.viewNotices || "View All →"}
              </Link>
            </div>
            {notices.length ? (
              <ul className="mt-4 divide-y divide-stone-100">
                {notices.slice(0, 4).map((notice) => (
                  <li key={notice.id} className="py-3 first:pt-0">
                    <p className="font-semibold text-stone-900 text-base">{notice.title}</p>
                    <p className="mt-1 text-sm text-stone-600 line-clamp-2">{notice.description}</p>
                    <p className="mt-2 text-xs font-medium text-stone-400">{notice.createdDate}</p>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="mt-4 rounded-xl bg-stone-50 px-4 py-8 text-center text-sm text-stone-500">
                {lang === "hi" ? "अभी कोई सूचना प्रकाशित नहीं हुई है।" : "No published notices yet."}
              </div>
            )}
          </div>

          <div className="card p-6 bg-white border border-stone-200 rounded-2xl shadow-sm">
            <h2 className="text-xl font-bold text-stone-900 border-b pb-4 flex items-center gap-2">
              <span>🗓️</span> {dict.home.upcomingEvents || "School Events"}
            </h2>
            {events.length ? (
              <ul className="mt-4 divide-y divide-stone-100">
                {events.slice(0, 4).map((event) => (
                  <li key={event.id} className="py-3 first:pt-0">
                    <p className="font-semibold text-stone-900">{event.title}</p>
                    <p className="mt-1 text-xs text-stone-500 font-medium">{event.startDate}{event.startTime ? ` · ${event.startTime}` : ""}</p>
                    {event.venueEn && <p className="mt-1 text-xs text-emerald-700 font-medium">📍 {event.venueEn}</p>}
                  </li>
                ))}
              </ul>
            ) : (
              <div className="mt-4 rounded-xl bg-stone-50 px-4 py-8 text-center text-sm text-stone-500">
                {lang === "hi" ? "कोई आगामी कार्यक्रम नहीं है।" : "No upcoming events yet."}
              </div>
            )}
          </div>
        </div>
      </section>

      {/* ---- Principal Desk & Message ------------------------------ */}
      <section className="mx-auto max-w-6xl px-4 py-6">
        <div className="card overflow-hidden bg-gradient-to-r from-stone-900 to-stone-800 text-white rounded-2xl shadow-lg grid md:grid-cols-3">
          <div className="p-6 md:p-8 md:col-span-2 flex flex-col justify-between">
            <div>
              <span className="text-xs font-bold uppercase tracking-wider text-emerald-400">Message from the Principal</span>
              <h3 className="mt-2 text-2xl font-bold text-white">Welcome to Bhagwan Das Inter College</h3>
              <p className="mt-4 text-stone-300 text-sm leading-relaxed">
                "Our commitment is to nurture curiosity, discipline, and moral values in every student from LKG through Class 12th. We provide an empowering environment that balances UP Board academic rigor with extracurricular talents."
              </p>
            </div>
            <div className="mt-6 pt-4 border-t border-stone-700">
              <p className="font-bold text-stone-100">{school.principalName || "Principal Desk"}</p>
              <p className="text-xs text-stone-400">Bhagwan Das Inter College, Ekauna</p>
            </div>
          </div>
          <div className="bg-stone-800 relative hidden md:block">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={SCHOOL_IMAGES.principal}
              alt="Principal"
              className="h-full w-full object-cover"
            />
          </div>
        </div>
      </section>

      {/* ---- Campus Facilities Showcase ---------------------------- */}
      <section className="mx-auto max-w-6xl px-4 py-12">
        <div className="text-center max-w-2xl mx-auto">
          <h2 className="text-2xl font-extrabold text-stone-900">Campus Facilities & Infrastructure</h2>
          <p className="text-sm text-stone-600 mt-1">Providing state-of-the-art facilities for academic and athletic development</p>
        </div>

        <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          <div className="rounded-2xl border border-stone-200 overflow-hidden bg-white shadow-sm hover:shadow-md transition">
            <div className="h-40 bg-stone-100 overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={SCHOOL_IMAGES.classroom} alt="Classrooms" className="h-full w-full object-cover" />
            </div>
            <div className="p-4">
              <h4 className="font-bold text-stone-900">Modern Classrooms</h4>
              <p className="text-xs text-stone-500 mt-1">Spacious, ventilated classrooms equipped for LKG to 12th learners.</p>
            </div>
          </div>

          <div className="rounded-2xl border border-stone-200 overflow-hidden bg-white shadow-sm hover:shadow-md transition">
            <div className="h-40 bg-stone-100 overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={SCHOOL_IMAGES.labs} alt="Science & Computer Labs" className="h-full w-full object-cover" />
            </div>
            <div className="p-4">
              <h4 className="font-bold text-stone-900">Science & IT Labs</h4>
              <p className="text-xs text-stone-500 mt-1">Well-equipped Physics, Chemistry, Biology & Computer laboratories.</p>
            </div>
          </div>

          <div className="rounded-2xl border border-stone-200 overflow-hidden bg-white shadow-sm hover:shadow-md transition">
            <div className="h-40 bg-stone-100 overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={SCHOOL_IMAGES.library} alt="School Library" className="h-full w-full object-cover" />
            </div>
            <div className="p-4">
              <h4 className="font-bold text-stone-900">Central Library</h4>
              <p className="text-xs text-stone-500 mt-1">Rich collection of course books, reference materials & journals.</p>
            </div>
          </div>

          <div className="rounded-2xl border border-stone-200 overflow-hidden bg-white shadow-sm hover:shadow-md transition">
            <div className="h-40 bg-stone-100 overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={SCHOOL_IMAGES.sports} alt="Playground & Sports" className="h-full w-full object-cover" />
            </div>
            <div className="p-4">
              <h4 className="font-bold text-stone-900">Sports & Athletics</h4>
              <p className="text-xs text-stone-500 mt-1">Playground for cricket, volleyball, athletics, and physical education.</p>
            </div>
          </div>
        </div>
      </section>

      {/* ---- Quick links -------------------------------------------- */}
      <section className="mx-auto max-w-6xl px-4 pb-12">
        <h2 className="text-xl font-bold text-stone-900 border-b pb-3">
          {dict.home.quickLinks || "Quick Navigation"}
        </h2>
        <ul className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
          {quickLinks.map((link) => (
            <li key={link.href}>
              <Link
                href={link.href}
                className="card flex flex-col items-center justify-center p-4 text-center rounded-2xl bg-white border border-stone-200 hover:border-emerald-500 hover:shadow-md transition group"
              >
                <span className="text-2xl mb-1 group-hover:scale-110 transition duration-200">{link.icon}</span>
                <span className="text-xs font-semibold text-stone-800">{link.label}</span>
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
