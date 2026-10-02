import CertificatesList from "../../../../components/portal/CertificatesList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.certificates || "Certificates" };
}

export default async function CertificatesPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <CertificatesList lang={lang} dict={dict} />;
}
