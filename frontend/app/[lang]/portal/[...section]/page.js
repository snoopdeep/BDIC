import { getDictionary } from "../../dictionaries";
import PortalSection from "../../../../components/PortalSection";

/**
 * A shared, data-aware route for portal sections. Each section supplies its
 * own API source through PortalSection, so the preview always shows records,
 * a genuine empty state, or an actionable error — never a fake placeholder.
 */
export async function generateMetadata({ params }) {
  const { lang, section } = await params;
  const dict = await getDictionary(lang);
  const key = sectionKey(section);
  return { title: dict.portal.sections[key] ?? dict.portal.title };
}

export default async function PortalSectionPage({ params }) {
  const { lang, section } = await params;
  const dict = await getDictionary(lang);

  const key = sectionKey(section);
  return <PortalSection section={key} lang={lang} dict={dict} />;
}

/**
 * Turns the URL segment back into the dictionary key for that section, so
 * /portal/study-material shows the heading for studyMaterial.
 */
function sectionKey(segments) {
  const first = Array.isArray(segments) ? segments[0] : segments;
  if (!first) {
    return "dashboard";
  }
  return String(first).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}
