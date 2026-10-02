import AuditLogView from "../../../../components/portal/AuditLogView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.auditLog };
}

export default async function AuditLogPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <AuditLogView lang={lang} dict={dict} />;
}
