import GalleryView from "../../../../components/portal/GalleryView";
import { getDictionary } from "../../dictionaries";

export async function generateMetadata({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return { title: dict.portal.sections.gallery || "Gallery" };
}

export default async function GalleryPage({ params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);
  return <GalleryView lang={lang} dict={dict} />;
}
