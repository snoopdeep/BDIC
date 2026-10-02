import Link from "next/link";
import { notFound } from "next/navigation";

import { getDictionary } from "../../dictionaries";
import {
  fetchPublicSchoolInfo,
  fetchPublicNotices,
  fetchPublicEvents,
  fetchPublicAchievements,
  fetchPublicFaculty,
  fetchPublicFacilities,
} from "../../../../lib/api";

const PAGE_CONTENT = {
  about: {
    en: [
      "Bhagwan Das Inter College serves learners from Ekauna and nearby villages with a focused UP Board education.",
      "The school provides quality education, holistic development, disciplined environment, and dedicated teaching staff for classes 6th to 12th.",
    ],
    hi: [
      "भगवान दास इंटर कॉलेज एकौना और आसपास के गांवों के विद्यार्थियों को उत्कृष्ट यूपी बोर्ड शिक्षा प्रदान करता है।",
      "विद्यालय कक्षा 6 से 12 तक गुणवत्तापूर्ण शिक्षा, सर्वांगीण विकास, अनुशासित वातावरण और समर्पित शिक्षण स्टाफ प्रदान करता है।",
    ],
  },
  academics: {
    en: [
      "The academic programme covers Classes 9–12 with Science, Arts, and Commerce streams.",
      "Strict compliance with UP Board curriculum, regular periodic tests, internal assignments, and board exam preparation.",
    ],
    hi: [
      "शैक्षणिक कार्यक्रम में विज्ञान, कला और वाणिज्य संकाय के साथ कक्षा 9 से 12 तक शामिल हैं।",
      "यूपी बोर्ड पाठ्यक्रम, नियमित आवधिक परीक्षाएं, आंतरिक असाइनमेंट और बोर्ड परीक्षा की तैयारी का पूर्ण पालन।",
    ],
  },
  admissions: {
    en: [
      "Admissions for the upcoming academic session are open. Families can submit online applications or visit the office.",
      "Required documents include previous marksheets, Transfer Certificate (TC), Aadhaar card, and passport-size photographs.",
    ],
    hi: [
      "आगामी शैक्षणिक सत्र के लिए प्रवेश खुले हैं। परिवार ऑनलाइन आवेदन जमा कर सकते हैं या विद्यालय कार्यालय आ सकते हैं।",
      "आवश्यक दस्तावेजों में पिछली अंकपत्र, स्थानांतरण प्रमाण पत्र (टीसी), आधार कार्ड और पासपोर्ट साइज फोटो शामिल हैं।",
    ],
  },
  contact: {
    en: [
      "School Address: Bhagwan Das Inter College, Ekauna, Chandauli, Uttar Pradesh - 232104",
      "Office Hours: Monday to Saturday, 8:00 AM – 2:00 PM",
    ],
    hi: [
      "विद्यालय का पता: भगवान दास इंटर कॉलेज, एकौना, चंदौली, उत्तर प्रदेश - 232104",
      "कार्यालय समय: सोमवार से शनिवार, प्रातः 8:00 से दोपहर 2:00 बजे तक",
    ],
  },
  privacy: {
    en: [
      "Student data, documents, and records are maintained with strict privacy controls in compliance with regulations.",
    ],
    hi: [
      "विद्यार्थी डेटा, दस्तावेज और रिकॉर्ड नियमों के अनुपालन में सख्त गोपनीयता नियंत्रण के साथ रखे जाते हैं।",
    ],
  },
  terms: {
    en: [
      "Users and portal account holders must adhere to appropriate usage, account security, and confidentiality.",
    ],
    hi: [
      "उपयोगकर्ताओं और पोर्टल खाताधारकों को उचित उपयोग, खाता सुरक्षा और गोपनीयता का पालन करना चाहिए।",
    ],
  },
};

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

function headingFor(dict, segments) {
  const first = Array.isArray(segments) ? segments[0] : segments;

  const key = TITLE_KEYS[first];
  if (key) {
    return dict.nav[key];
  }
  if (first === "privacy") {
    return dict.footer?.privacy || "Privacy Policy";
  }
  if (first === "terms") {
    return dict.footer?.terms || "Terms of Service";
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
    notFound();
  }

  const key = Array.isArray(slug) ? slug[0] : slug;
  let dynamicItems = null;
  let schoolInfo = null;

  try {
    if (key === "notices") {
      const res = await fetchPublicNotices({ timeoutMs: 3000 });
      dynamicItems = Array.isArray(res) ? res : res?.items || null;
    } else if (key === "events") {
      const res = await fetchPublicEvents({ timeoutMs: 3000 });
      dynamicItems = Array.isArray(res) ? res : res?.items || null;
    } else if (key === "faculty") {
      const res = await fetchPublicFaculty({ timeoutMs: 3000 });
      dynamicItems = Array.isArray(res) ? res : res?.items || null;
    } else if (key === "facilities") {
      const res = await fetchPublicFacilities({ timeoutMs: 3000 });
      dynamicItems = Array.isArray(res) ? res : res?.items || null;
    } else if (key === "achievements") {
      const res = await fetchPublicAchievements({ timeoutMs: 3000 });
      dynamicItems = Array.isArray(res) ? res : res?.items || null;
    } else if (key === "about" || key === "contact") {
      schoolInfo = await fetchPublicSchoolInfo({ timeoutMs: 3000 });
    }
  } catch (_err) {
    // Graceful fallback to static copy if API unreachable
  }

  const content = PAGE_CONTENT[key];
  const paragraphs = content?.[lang] ?? content?.en ?? [];

  return (
    <div className="mx-auto max-w-4xl px-4 py-12">
      <h1 className="text-3xl font-bold text-brand-900">{heading}</h1>

      {schoolInfo && key === "contact" && (
        <div className="card mt-6 p-6 space-y-3 bg-white border border-stone-200 rounded-lg shadow-sm">
          <p className="font-semibold text-lg text-stone-900">
            {lang === "hi" ? schoolInfo.nameHi : schoolInfo.nameEn}
          </p>
          <p className="text-sm text-stone-600">
            📍 {lang === "hi" ? schoolInfo.addressHi : schoolInfo.addressEn}
          </p>
          <p className="text-sm text-stone-600">
            📞 {schoolInfo.phonePrimary} {schoolInfo.phoneSecondary ? `/ ${schoolInfo.phoneSecondary}` : ""}
          </p>
          <p className="text-sm text-stone-600">✉️ {schoolInfo.email}</p>
          <p className="text-sm text-stone-600">
            ⏰ {lang === "hi" ? schoolInfo.officeHoursHi : schoolInfo.officeHoursEn}
          </p>
        </div>
      )}

      {dynamicItems && dynamicItems.length > 0 ? (
        <div className="mt-6 space-y-4">
          {dynamicItems.map((item, idx) => (
            <div key={item.id || idx} className="card p-5 border rounded-lg bg-white shadow-sm">
              <h3 className="font-bold text-lg text-stone-900">
                {item.title || item.name || item.heading || "—"}
              </h3>
              {(item.description || item.body || item.summary) && (
                <p className="mt-2 text-stone-700 text-sm">
                  {item.description || item.body || item.summary}
                </p>
              )}
              {item.publishedAt && (
                <p className="mt-3 text-xs text-stone-400">
                  {new Date(item.publishedAt).toLocaleDateString()}
                </p>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="card mt-6 space-y-4 p-6 text-zinc-700 bg-white border rounded-lg shadow-sm">
          {paragraphs.map((paragraph) => (
            <p key={paragraph}>{paragraph}</p>
          ))}
        </div>
      )}

      <p className="mt-8">
        <Link
          href={`/${lang}`}
          className="font-medium text-brand-700 underline-offset-2 hover:underline inline-flex items-center gap-1"
        >
          ← {dict.nav.home}
        </Link>
      </p>
    </div>
  );
}
