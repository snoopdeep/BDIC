import StudyMaterialList from "../../../../components/portal/StudyMaterialList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.studyMaterial || "Study Material" };
}

export default async function StudyMaterialPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <StudyMaterialList lang={lang} dict={dict} />;
}
