import SettingsPanel from "../../../../components/portal/SettingsPanel";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.settings || "Settings" };
}

export default async function SettingsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <SettingsPanel lang={lang} dict={dict} />;
}
