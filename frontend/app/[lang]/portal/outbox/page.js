import OutboxView from "../../../../components/portal/OutboxView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.outbox || "Outbox" };
}

export default async function OutboxPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <OutboxView lang={lang} dict={dict} />;
}
