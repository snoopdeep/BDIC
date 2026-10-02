import Link from "next/link";

import { pick } from "../lib/school";

/**
 * The public footer.
 *
 * The school's address, phone and office hours are here on every page, because
 * that is what a parent on a phone is most often looking for. Fields the school
 * has not provided yet are simply left out rather than shown as a blank label.
 */
export default function SiteFooter({ lang, dict, school }) {
  const schoolName = pick(school, "name", lang);
  const address = pick(school, "address", lang);
  const officeHours = pick(school, "officeHours", lang);

  const socialLinks = [
    { href: school.instagramUrl, label: "Instagram" },
    { href: school.facebookUrl, label: "Facebook" },
    { href: school.googlePlaceUrl, label: "Google" },
  ].filter((link) => Boolean(link.href));

  return (
    <footer className="mt-16 border-t border-brand-100 bg-white">
      <div className="mx-auto grid max-w-6xl gap-8 px-4 py-10 sm:grid-cols-2 lg:grid-cols-4">
        <div className="sm:col-span-2 lg:col-span-1">
          <p className="text-base font-semibold text-brand-900">{schoolName}</p>
          <p className="mt-1 text-sm text-zinc-600">{school.board}</p>
          {school.affiliationNo ? (
            <p className="mt-1 text-sm text-zinc-500">
              {school.affiliationNo}
            </p>
          ) : null}
        </div>

        <div>
          <h2 className="text-sm font-semibold text-zinc-900">
            {dict.footer.address}
          </h2>
          <address className="mt-2 text-sm not-italic leading-relaxed text-zinc-600">
            {address}
            {school.pincode ? <> — {school.pincode}</> : null}
          </address>
          {officeHours ? (
            <p className="mt-3 text-sm text-zinc-600">
              <span className="font-medium text-zinc-800">
                {dict.footer.officeHours}:{" "}
              </span>
              {officeHours}
            </p>
          ) : null}
        </div>

        <div>
          <h2 className="text-sm font-semibold text-zinc-900">
            {dict.footer.phone}
          </h2>
          {school.phonePrimary ? (
            <p className="mt-2 text-sm">
              <a
                href={`tel:${school.phonePrimary}`}
                className="text-brand-700 underline-offset-2 hover:underline"
              >
                {school.phonePrimary}
              </a>
            </p>
          ) : (
            <p className="mt-2 text-sm text-zinc-400">—</p>
          )}
          {school.phoneSecondary ? (
            <p className="mt-1 text-sm">
              <a
                href={`tel:${school.phoneSecondary}`}
                className="text-brand-700 underline-offset-2 hover:underline"
              >
                {school.phoneSecondary}
              </a>
            </p>
          ) : null}
          {school.email ? (
            <p className="mt-3 text-sm">
              <span className="block font-medium text-zinc-800">
                {dict.footer.email}
              </span>
              <a
                href={`mailto:${school.email}`}
                className="break-all text-brand-700 underline-offset-2 hover:underline"
              >
                {school.email}
              </a>
            </p>
          ) : null}
        </div>

        <div>
          <h2 className="text-sm font-semibold text-zinc-900">
            {dict.footer.followUs}
          </h2>
          <ul className="mt-2 space-y-1 text-sm">
            {socialLinks.map((link) => (
              <li key={link.label}>
                <a
                  href={link.href}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="text-brand-700 underline-offset-2 hover:underline"
                >
                  {link.label}
                </a>
              </li>
            ))}
          </ul>

          <ul className="mt-4 space-y-1 text-sm">
            <li>
              <Link
                href={`/${lang}/privacy`}
                className="text-zinc-600 underline-offset-2 hover:underline"
              >
                {dict.footer.privacy}
              </Link>
            </li>
            <li>
              <Link
                href={`/${lang}/terms`}
                className="text-zinc-600 underline-offset-2 hover:underline"
              >
                {dict.footer.terms}
              </Link>
            </li>
          </ul>
        </div>
      </div>

      <div className="border-t border-brand-100 px-4 py-4">
        <p className="mx-auto max-w-6xl text-xs text-zinc-500">
          © {new Date().getFullYear()} {schoolName}. {dict.footer.rightsReserved}
        </p>
      </div>
    </footer>
  );
}
