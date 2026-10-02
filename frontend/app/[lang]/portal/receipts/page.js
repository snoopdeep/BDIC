import ReceiptsList from "../../../../components/portal/ReceiptsList";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.receipts };
}

export default async function ReceiptsPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <ReceiptsList lang={lang} dict={dict} />;
}
