import StudentsList from "../../../../components/portal/StudentsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.students };
}

export default async function StudentsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <StudentsList lang={lang} dict={dict} />;
}
