import Link from "next/link";

import LanguageToggle from "./LanguageToggle";
import MobileNav from "./MobileNav";
import { pick } from "../lib/school";

/**
 * The public site's header.
 *
 * Rendered on the server, so the school's name and the navigation are in the
 * first HTML the browser receives. Only the language switch and the mobile
 * menu are client components, which keeps the marketing pages static.
 */
export default function SiteHeader({ lang, dict, school }) {
  const navItems = [
    { href: `/${lang}`, label: dict.nav.home },
    { href: `/${lang}/about`, label: dict.nav.about },
    { href: `/${lang}/academics`, label: dict.nav.academics },
    { href: `/${lang}/admissions`, label: dict.nav.admissions },
    { href: `/${lang}/notices`, label: dict.nav.notices },
    { href: `/${lang}/gallery`, label: dict.nav.gallery },
    { href: `/${lang}/contact`, label: dict.nav.contact },
  ];

  const schoolName = pick(school, "name", lang);
  const schoolAddress = pick(school, "address", lang);

  // Note: backdrop-blur-sm, not backdrop-blur. Tailwind v4 renamed the bare
  // utility, and the old name silently generates nothing.
  return (
    <header className="sticky top-0 z-40 border-b border-brand-100 bg-white/95 backdrop-blur-sm">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
        <Link
          href={`/${lang}`}
          className="flex min-w-0 items-center gap-3"
          aria-label={schoolName}
        >
          {/* TODO school: replace this monogram with the school's real logo. */}
          <span
            aria-hidden="true"
            className="grid size-11 shrink-0 place-items-center rounded-xl bg-brand-700 text-sm font-bold text-white"
          >
            BD
          </span>
          <span className="min-w-0">
            <span className="block truncate text-base font-semibold leading-tight text-brand-900 sm:text-lg">
              {schoolName}
            </span>
            <span className="block truncate text-xs text-zinc-500">
              {schoolAddress}
            </span>
          </span>
        </Link>

        <nav
          aria-label={dict.nav.menu}
          className="ml-auto hidden items-center gap-1 lg:flex"
        >
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="rounded-lg px-3 py-2 text-sm font-medium text-zinc-700 transition hover:bg-brand-50 hover:text-brand-800"
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-2 lg:ml-2">
          <LanguageToggle
            locale={lang}
            otherLocale={dict.meta.otherLocale}
            label={dict.meta.otherLocaleName}
          />
          <Link
            href={`/${lang}/login`}
            className="hidden rounded-lg bg-brand-700 px-4 py-2 text-sm font-semibold text-white transition hover:bg-brand-800 sm:inline-flex sm:items-center"
          >
            {dict.nav.portalLogin}
          </Link>
          {/* Only the four strings it needs, not the whole dictionary: a
              client component's props are serialised into the page, and
              shipping all 152 translations to every visitor is waste. */}
          <MobileNav
            items={navItems}
            lang={lang}
            labels={{
              menu: dict.nav.menu,
              openMenu: dict.nav.openMenu,
              closeMenu: dict.nav.closeMenu,
              portalLogin: dict.nav.portalLogin,
            }}
          />
        </div>
      </div>
    </header>
  );
}
