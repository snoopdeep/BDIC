import NoticesList from "../../../../components/portal/NoticesList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.notices || "Notices" };
}

export default async function NoticesPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <NoticesList lang={lang} dict={dict} />;
}
