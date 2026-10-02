import SiteFooter from "../../../components/SiteFooter";
import SiteHeader from "../../../components/SiteHeader";
import { getDictionary } from "../dictionaries";
import { getSchool } from "../../../lib/school";

/**
 * Chrome for the public website.
 *
 * A route group, so it wraps the public pages without adding anything to the
 * URL. The portal and the sign-in page sit outside it and have their own
 * chrome, because a parent checking fees does not need the marketing
 * navigation above them.
 */
export default async function SiteLayout({ children, params }) {
  const { lang } = await params;
  const [dict, { school, apiReachable }] = await Promise.all([
    getDictionary(lang),
    getSchool(),
  ]);

  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader lang={lang} dict={dict} school={school} />

      {/*
        A build-time notice, not a production one. If the frontend is running
        and the Go API is not, every page still renders from the fallback
        values — and this line says why the content looks thin, instead of
        leaving the reader to guess.
      */}
      {!apiReachable && process.env.NODE_ENV !== "production" ? (
        <p className="bg-warn-100 px-4 py-2 text-center text-sm text-warn-700">
          The API is not reachable at {process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}.
          Run <code className="font-mono">make api</code> in another terminal.
          The page below is showing fallback content.
        </p>
      ) : null}

      <main id="main" className="flex-1">
        {children}
      </main>

      <SiteFooter lang={lang} dict={dict} school={school} />
    </div>
  );
}
