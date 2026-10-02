import TimetableView from "../../../../components/portal/TimetableView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.timetable };
}

export default async function TimetablePage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <TimetableView lang={lang} dict={dict} />;
}
