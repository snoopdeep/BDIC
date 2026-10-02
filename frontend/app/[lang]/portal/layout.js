import PortalShell from "../../../components/PortalShell";
import { getDictionary } from "../dictionaries";

export const metadata = {
  // The portal holds student records. It should never be indexed.
  robots: { index: false, follow: false },
};

/**
 * The portal's layout.
 *
 * The dictionary is loaded on the server and handed to the shell as a prop, so
 * only one language's strings reach the browser rather than both.
 */
export default async function PortalLayout({ children, params }) {
  const { lang } = await params;
  const dict = await getDictionary(lang);

  return (
    <PortalShell lang={lang} dict={dict}>
      {children}
    </PortalShell>
  );
}
