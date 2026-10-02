import "../globals.css";
import { DEFAULT_LOCALE, LOCALES, getDictionary, hasLocale } from "./dictionaries";
import { getSchool, pick } from "../../lib/school";

/**
 * The root layout.
 *
 * It lives under [lang] rather than at the top of app/, which is what makes
 * `lang` available to every page and layout in the system. There is no
 * app/layout.js: this is it.
 */

export async function generateStaticParams() {
  // Both languages are prerendered at build time, so the public pages are
  // static HTML in Hindi and in English.
  return LOCALES.map((lang) => ({ lang }));
}

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const locale = hasLocale(lang) ? lang : DEFAULT_LOCALE;
  const [dict, { school }] = await Promise.all([
    getDictionary(locale),
    getSchool(),
  ]);

  const name = pick(school, "name", locale);
  const address = pick(school, "address", locale);

  return {
    title: {
      default: name,
      template: `%s — ${name}`,
    },
    description: `${name}, ${address}. ${school.board}. ${dict.home.tagline}`,
    // Tells search engines that the two languages are the same page, so the
    // school is not competing against itself in results.
    alternates: {
      canonical: `/${locale}`,
      languages: {
        hi: "/hi",
        en: "/en",
      },
    },
    openGraph: {
      title: name,
      description: dict.home.tagline,
      locale: locale === "hi" ? "hi_IN" : "en_IN",
      type: "website",
      siteName: name,
    },
    // Enough for Google to show the school properly in Search and on Maps.
    other: {
      "geo.region": "IN-UP",
      "geo.placename": school.district || "Chandauli",
    },
  };
}

export const viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#4338ca",
};

export default async function RootLayout({ children, params }) {
  const { lang } = await params;

  // Deliberately not notFound() here. A not-found boundary lives in the parent
  // segment, and this layout IS the root — so throwing from here would bypass
  // app/[lang]/not-found.js and land on Next's own English 404. An unknown
  // language falls back instead, and the pages below handle a path that does
  // not exist.
  const locale = hasLocale(lang) ? lang : DEFAULT_LOCALE;

  const [dict, { school }] = await Promise.all([
    getDictionary(locale),
    getSchool(),
  ]);

  const schoolName = pick(school, "name", locale);

  // Structured data, so a search for the school shows its address, phone and
  // opening hours rather than only a blue link.
  const structuredData = {
    "@context": "https://schema.org",
    "@type": "School",
    name: schoolName,
    alternateName: pick(school, "name", locale === "hi" ? "en" : "hi"),
    address: {
      "@type": "PostalAddress",
      streetAddress: school.village || undefined,
      addressLocality: school.district || undefined,
      addressRegion: school.state || undefined,
      postalCode: school.pincode || undefined,
      addressCountry: "IN",
    },
    telephone: school.phonePrimary || undefined,
    email: school.email || undefined,
    sameAs: [
      school.instagramUrl,
      school.facebookUrl,
      school.googlePlaceUrl,
    ].filter(Boolean),
  };

  return (
    <html lang={locale} data-scroll-behavior="smooth">
      <body className="min-h-dvh antialiased">
        {/*
          No manual <head>. React 19 hoists <link rel="stylesheet"> and
          <script> into the document head from anywhere in the tree, and Next's
          metadata API is the only other thing writing there — which is what
          keeps the two from fighting over de-duplication.

          The font comes from Google Fonts rather than next/font so a build
          with no network still succeeds. Devanagari conjuncts break visibly in
          the default system stacks on Windows and older Android, which is most
          of this school's audience, so a real Devanagari face is not optional.
        */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin="anonymous"
        />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;500;600;700&family=Noto+Sans+Devanagari:wght@400;500;600;700&display=swap"
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: serialiseJsonLd(structuredData) }}
        />

        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:px-4 focus:py-2 focus:text-brand-800 focus:shadow-sm"
        >
          {dict.common.skipToContent}
        </a>
        {children}
      </body>
    </html>
  );
}

/**
 * Serialises structured data for embedding in a <script> tag.
 *
 * JSON.stringify does not escape "<", and this object is built from the
 * school's own database rows — a name or address the office typed. If one ever
 * contained "</script>", a raw stringify would close the element and whatever
 * followed would run as markup. Escaping the angle bracket costs nothing.
 */
function serialiseJsonLd(value) {
  return JSON.stringify(value).replace(/</g, "\\u003c");
}
