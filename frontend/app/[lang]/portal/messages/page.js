import MessagesList from "../../../../components/portal/MessagesList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.messages || "Messages" };
}

export default async function MessagesPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <MessagesList lang={lang} dict={dict} />;
}
