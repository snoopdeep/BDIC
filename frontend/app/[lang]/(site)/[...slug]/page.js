import Link from "next/link";
import { notFound } from "next/navigation";

import { getDictionary } from "../../dictionaries";

// Demonstration content keeps every public navigation route useful in a local
// review. The school will replace these sample paragraphs with approved copy
// through the site-content module before launch.
const PAGE_CONTENT = {
  about: {
    en: ["Bhagwan Das Inter College serves learners from Ekauna and nearby villages with a focused UP Board education.", "This demonstration page shows the layout for the school story, leadership message, values, and campus highlights."],
    hi: ["भगवान दास इंटर कॉलेज एकौना और आसपास के गांवों के विद्यार्थियों को यूपी बोर्ड शिक्षा प्रदान करता है।", "यह प्रदर्शन पृष्ठ विद्यालय परिचय, नेतृत्व संदेश, मूल्यों और परिसर की जानकारी के लिए बनाया गया है।"],
  },
  academics: {
    en: ["The academic programme covers Classes 9–12 with subject and stream choices recorded in the school portal.", "Use the portal timetable, study material, homework, examinations, and report-card areas to review the complete academic flow."],
    hi: ["शैक्षणिक कार्यक्रम में कक्षा 9 से 12 तक की कक्षाएं, विषय और स्ट्रीम विकल्प शामिल हैं।", "पूरा शैक्षणिक प्रवाह देखने के लिए पोर्टल में समय-सारिणी, अध्ययन सामग्री, गृहकार्य, परीक्षा और रिपोर्ट कार्ड देखें।"],
  },
  admissions: {
    en: ["Admissions are demonstrated for the current academic session. Families can review the required documents and submit an application online.", "The office verifies documents, records the decision, creates the student profile, and can issue a portal account."],
    hi: ["वर्तमान शैक्षणिक सत्र के लिए प्रवेश प्रक्रिया का प्रदर्शन उपलब्ध है। परिवार आवश्यक दस्तावेज देख सकते हैं और ऑनलाइन आवेदन कर सकते हैं।", "कार्यालय दस्तावेज सत्यापित करता है, निर्णय दर्ज करता है, विद्यार्थी प्रोफ़ाइल बनाता है और पोर्टल खाता जारी कर सकता है।"],
  },
  notices: {
    en: ["Published school notices appear here and in the signed-in notification centre.", "Sample notices are included in the local demo data so the table, audience rules, and acknowledgement flow can be reviewed."],
    hi: ["प्रकाशित विद्यालय सूचनाएं यहां और साइन-इन सूचना केंद्र में दिखाई देती हैं।", "स्थानीय डेमो डेटा में नमूना सूचनाएं शामिल हैं ताकि सूची, दर्शक नियम और स्वीकृति प्रक्रिया देखी जा सके।"],
  },
  events: {
    en: ["This page presents upcoming examinations, parent meetings, cultural programmes, and sports events.", "All dates in the local build are sample data and must be replaced with the school calendar before launch."],
    hi: ["यह पृष्ठ आगामी परीक्षाएं, अभिभावक बैठकें, सांस्कृतिक कार्यक्रम और खेल गतिविधियां दिखाता है।", "स्थानीय बिल्ड की सभी तिथियां नमूना डेटा हैं और लॉन्च से पहले विद्यालय कैलेंडर से बदली जानी चाहिए।"],
  },
  gallery: {
    en: ["The gallery is ready for approved, consent-checked campus and event photographs.", "No real student photographs are included in this demo; upload only school-approved images after consent is recorded."],
    hi: ["गैलरी अनुमोदित और सहमति-युक्त परिसर व कार्यक्रम फोटोग्राफ के लिए तैयार है।", "इस डेमो में वास्तविक विद्यार्थियों की तस्वीरें शामिल नहीं हैं; केवल विद्यालय-अनुमोदित चित्र ही अपलोड करें।"],
  },
  achievements: {
    en: ["Academic, sports, cultural, scholarship, and school achievements can be featured here.", "The local sample data demonstrates how achievement titles, categories, ranks, and descriptions are presented."],
    hi: ["शैक्षणिक, खेल, सांस्कृतिक, छात्रवृत्ति और विद्यालय उपलब्धियां यहां दिखाई जा सकती हैं।", "स्थानीय नमूना डेटा शीर्षक, श्रेणी, रैंक और विवरण का प्रदर्शन करता है।"],
  },
  faculty: {
    en: ["Faculty profiles are published only when the school enables website visibility for that staff member.", "The demo uses fictional staff information and contains no personal information about the school team."],
    hi: ["स्टाफ प्रोफ़ाइल तभी प्रकाशित होती है जब विद्यालय उस सदस्य के लिए वेबसाइट दृश्यता सक्षम करे।", "डेमो में काल्पनिक स्टाफ जानकारी है और विद्यालय टीम की व्यक्तिगत जानकारी शामिल नहीं है।"],
  },
  facilities: {
    en: ["This section is for classrooms, laboratories, library, computer resources, sports, and student-support facilities.", "The final page will use school-approved descriptions and photographs."],
    hi: ["यह अनुभाग कक्षाओं, प्रयोगशालाओं, पुस्तकालय, कंप्यूटर संसाधनों, खेल और विद्यार्थी सहायता सुविधाओं के लिए है।", "अंतिम पृष्ठ में विद्यालय-अनुमोदित विवरण और फोटोग्राफ शामिल होंगे।"],
  },
  downloads: {
    en: ["Admission forms, syllabi, fee information, policies, and previous papers can be shared as controlled downloads.", "The demo intentionally contains no invented official documents."],
    hi: ["प्रवेश फॉर्म, पाठ्यक्रम, शुल्क जानकारी, नीतियां और पिछले प्रश्नपत्र नियंत्रित डाउनलोड के रूप में साझा किए जा सकते हैं।", "डेमो में कोई मनगढ़ंत आधिकारिक दस्तावेज शामिल नहीं है।"],
  },
  contact: {
    en: ["Contact details, office hours, map, and social links are shown in the site footer and school profile.", "Replace demo contact values with confirmed office details before the public site is launched."],
    hi: ["संपर्क विवरण, कार्यालय समय, मानचित्र और सोशल लिंक साइट फ़ुटर व विद्यालय प्रोफ़ाइल में दिखाए जाते हैं।", "सार्वजनिक साइट लॉन्च से पहले डेमो संपर्क विवरण को पुष्टि किए गए कार्यालय विवरण से बदलें।"],
  },
  privacy: {
    en: ["This demonstration privacy page explains that student data, documents, and photographs are available only to authorised users.", "The school should approve its final retention, consent, grievance, and contact policy before publication."],
    hi: ["यह डेमो गोपनीयता पृष्ठ बताता है कि विद्यार्थी डेटा, दस्तावेज और फोटो केवल अधिकृत उपयोगकर्ताओं के लिए उपलब्ध हैं।", "प्रकाशन से पहले विद्यालय को अंतिम डेटा-धारण, सहमति, शिकायत और संपर्क नीति स्वीकृत करनी चाहिए।"],
  },
  terms: {
    en: ["This demonstration terms page covers appropriate portal use, account security, and the need to keep login details private.", "The school should approve final portal terms before opening accounts to families."],
    hi: ["यह डेमो नियम पृष्ठ उचित पोर्टल उपयोग, खाता सुरक्षा और लॉगिन विवरण गोपनीय रखने की आवश्यकता बताता है।", "परिवारों के लिए खाते खोलने से पहले विद्यालय को अंतिम पोर्टल नियम स्वीकृत करने चाहिए।"],
  },
};

/** Maps a URL path to the dictionary key that names it, for the heading. */
const TITLE_KEYS = {
  about: "about",
  academics: "academics",
  admissions: "admissions",
  notices: "notices",
  events: "events",
  gallery: "gallery",
  achievements: "achievements",
  faculty: "faculty",
  facilities: "facilities",
  downloads: "downloads",
  contact: "contact",
};

/**
 * Returns the heading for a supported public path, or null for anything else.
 *
 * Null matters: without it this catch-all answers 200 "Coming soon" for every
 * URL under a language, so nothing on the site could ever 404 — a typo, a
 * broken external link, and a real page would all look the same to a visitor
 * and to a search engine.
 */
function headingFor(dict, segments) {
  const first = Array.isArray(segments) ? segments[0] : segments;

  const key = TITLE_KEYS[first];
  if (key) {
    return dict.nav[key];
  }
  if (first === "privacy") {
    return dict.footer.privacy;
  }
  if (first === "terms") {
    return dict.footer.terms;
  }
  return null;
}

export async function generateMetadata({ params }) {
  const { lang, slug } = await params;
  const dict = await getDictionary(lang);
  const heading = headingFor(dict, slug);
  return {
    title: heading ?? dict.common.schoolShortName,
    robots: { index: true, follow: true },
  };
}

export default async function SitePage({ params }) {
  const { lang, slug } = await params;
  const dict = await getDictionary(lang);

  const heading = headingFor(dict, slug);
  if (!heading) {
    // A real 404, with the right status code, rendered by
    // app/[lang]/not-found.js.
    notFound();
  }
  const key = Array.isArray(slug) ? slug[0] : slug;
  const content = PAGE_CONTENT[key];
  const paragraphs = content?.[lang] ?? content?.en ?? [];

  return (
    <div className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="text-3xl font-bold text-brand-900">{heading}</h1>

      <div className="card mt-6 space-y-4 p-6 text-zinc-700">
        {paragraphs.map((paragraph) => (
          <p key={paragraph}>{paragraph}</p>
        ))}
      </div>

      <p className="mt-6">
        <Link
          href={`/${lang}`}
          className="font-medium text-brand-700 underline-offset-2 hover:underline"
        >
          {dict.nav.home}
        </Link>
      </p>
    </div>
  );
}
