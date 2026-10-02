import HomeworkList from "../../../../components/portal/HomeworkList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.homework || "Homework" };
}

export default async function HomeworkPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <HomeworkList lang={lang} dict={dict} />;
}
