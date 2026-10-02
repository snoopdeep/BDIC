import ReportCardView from "../../../../components/portal/ReportCardView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.reportCards || "Report Cards" };
}

export default async function ReportCardsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <ReportCardView lang={lang} dict={dict} />;
}
