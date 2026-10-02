import ExamsList from "../../../../components/portal/ExamsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.exams || "Exams" };
}

export default async function ExamsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <ExamsList lang={lang} dict={dict} />;
}
