import FeesList from "../../../../components/portal/FeesList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.fees };
}

export default async function FeesPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <FeesList lang={lang} dict={dict} />;
}
