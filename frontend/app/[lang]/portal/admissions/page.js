import AdmissionsList from "../../../../components/portal/AdmissionsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.admissions || "Admissions" };
}

export default async function AdmissionsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <AdmissionsList lang={lang} dict={dict} />;
}
