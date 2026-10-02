import PortalDashboard from "../../../components/PortalDashboard";
import { getDictionary } from "../dictionaries";

// The catch-all portal route handles named sections such as /portal/fees, but
// it cannot match the portal root itself. This page is the post-sign-in landing
// route used by LoginPanel.
export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.title };
}

export default async function PortalHome({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <PortalDashboard lang={lang} dict={dict} />;
}
