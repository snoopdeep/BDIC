import ResultsList from "../../../../components/portal/ResultsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.results || "Results" };
}

export default async function ResultsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <ResultsList lang={lang} dict={dict} />;
}
