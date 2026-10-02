import AttendanceView from "../../../../components/portal/AttendanceView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.attendance };
}

export default async function AttendancePage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <AttendanceView lang={lang} dict={dict} />;
}
