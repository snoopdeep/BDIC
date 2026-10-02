import Link from "next/link";

import LanguageToggle from "../../../components/LanguageToggle";
import LoginPanel from "../../../components/LoginPanel";
import { getDictionary } from "../dictionaries";
import { getSchool, pick } from "../../../lib/school";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return {
    title: dict.login.title,
    // Nothing here is worth indexing, and a sign-in page in search results is
    // only ever a phishing target.
    robots: { index: false, follow: false },
  };
}

/**
 * The sign-in page.
 *
 * Outside the (site) route group on purpose: a parent who has come here to
 * check a fee balance does not need the marketing navigation, and a page with
 * one job is a page that is hard to get wrong.
 */
export default async function LoginPage({ params }) {
  const { lang } = await params;
  const [dict, { school }] = await Promise.all([
    getDictionary(lang),
    getSchool(),
  ]);

  return (
    <div className="flex min-h-dvh flex-col bg-brand-50">
      <header className="border-b border-brand-100 bg-white">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
          <Link href={`/${lang}`} className="flex min-w-0 items-center gap-3">
            <span
              aria-hidden="true"
              className="grid size-10 shrink-0 place-items-center rounded-xl bg-brand-700 text-sm font-bold text-white"
            >
              BD
            </span>
            <span className="truncate text-base font-semibold text-brand-900">
              {pick(school, "name", lang)}
            </span>
          </Link>
          <div className="ml-auto">
            <LanguageToggle
              locale={lang}
              otherLocale={dict.meta.otherLocale}
              label={dict.meta.otherLocaleName}
            />
          </div>
        </div>
      </header>

      <main
        id="main"
        className="flex flex-1 items-start justify-center px-4 py-10 sm:items-center"
      >
        <LoginPanel lang={lang} dict={dict} />
      </main>
    </div>
  );
}
