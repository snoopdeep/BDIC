import { NextResponse } from "next/server";

/**
 * Locale routing.
 *
 * Every page lives under /[lang]/..., so a request to a bare path has to be
 * sent to one of the two languages. This file is called proxy.js rather than
 * middleware.js: Next.js 16 renamed the convention.
 *
 * The order of preference is deliberate:
 *   1. The language the visitor last chose, remembered in a cookie.
 *   2. The language their browser asks for.
 *   3. Hindi, because this is a village school in Uttar Pradesh and Hindi is
 *      what most parents read. TODO school: confirm this default.
 */

const LOCALES = ["hi", "en"];
const DEFAULT_LOCALE = "hi";
const LOCALE_COOKIE = "bdic_locale";

function preferredLocale(request) {
  const remembered = request.cookies.get(LOCALE_COOKIE)?.value;
  if (LOCALES.includes(remembered)) {
    return remembered;
  }

  // Accept-Language looks like "hi-IN,hi;q=0.9,en-US;q=0.8". Read it in the
  // order the browser gave and take the first language we actually have,
  // rather than pulling in a full locale-negotiation library for two options.
  const header = request.headers.get("accept-language");
  if (header) {
    for (const part of header.split(",")) {
      const tag = part.split(";")[0].trim().toLowerCase();
      const base = tag.split("-")[0];
      if (LOCALES.includes(base)) {
        return base;
      }
    }
  }

  return DEFAULT_LOCALE;
}

export function proxy(request) {
  const { pathname } = request.nextUrl;

  const hasLocale = LOCALES.some(
    (locale) => pathname === `/${locale}` || pathname.startsWith(`/${locale}/`),
  );
  if (hasLocale) {
    return NextResponse.next();
  }

  const locale = preferredLocale(request);
  const url = request.nextUrl.clone();
  // "/" becomes "/hi", not "/hi/".
  url.pathname = pathname === "/" ? `/${locale}` : `/${locale}${pathname}`;

  const response = NextResponse.redirect(url);

  /*
    This redirect's destination depends on the visitor's cookie and their
    Accept-Language header. Without Vary, a shared cache could serve one
    visitor's "/ → /hi" to the next visitor, who asked for English.
  */
  response.headers.set("Vary", "Accept-Language, Cookie");

  // Remember what we negotiated, so the next visit skips the guessing and the
  // choice survives even if the visitor never touches the language switch.
  response.cookies.set(LOCALE_COOKIE, locale, {
    path: "/",
    maxAge: 60 * 60 * 24 * 365,
    sameSite: "lax",
  });

  return response;
}

export const config = {
  /*
    Without a matcher this would run on every request, including the CSS, the
    JavaScript bundles, and everything in public/ — which would rewrite their
    paths and leave the site unstyled.

    Each exclusion is anchored with (?:$|/) so it matches a whole path segment.
    A bare (?!api) would also exclude a future /apiary page, which is the kind
    of bug nobody finds for a year.
  */
  matcher: [
    "/((?!_next(?:$|/)|api(?:$|/)|favicon\\.ico$|robots\\.txt$|sitemap\\.xml$|manifest\\.webmanifest$|.*\\.[\\w]+$).*)",
  ],
};
