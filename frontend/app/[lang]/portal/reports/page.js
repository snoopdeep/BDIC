import ReportsList from "../../../../components/portal/ReportsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.reports || "Reports" };
}

export default async function ReportsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <ReportsList lang={lang} dict={dict} />;
}
