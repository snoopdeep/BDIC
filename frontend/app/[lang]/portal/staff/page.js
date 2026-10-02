import StaffList from "../../../../components/portal/StaffList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.staff };
}

export default async function StaffPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <StaffList lang={lang} dict={dict} />;
}
