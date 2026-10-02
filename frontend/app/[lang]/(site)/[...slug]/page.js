import Link from "next/link";
import { notFound } from "next/navigation";

import { getDictionary } from "../../dictionaries";
import {
  fetchPublicSchool,
  fetchPublicNotices,
  fetchPublicEvents,
  fetchPublicAchievements,
  fetchPublicFaculty,
  fetchPublicFacilities,
} from "../../../../lib/api";

const DEMO_GALLERY_IMAGES = [
  { id: "1", title: "School Building Front View", category: "Campus", url: "https://images.unsplash.com/photo-1562774053-701939374585?auto=format&fit=crop&w=600&q=80" },
  { id: "2", title: "Annual Sports Day Competition", category: "Sports", url: "https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=600&q=80" },
  { id: "3", title: "Science Practical Lab Work", category: "Academics", url: "https://images.unsplash.com/photo-1532094349884-543bc11b234d?auto=format&fit=crop&w=600&q=80" },
  { id: "4", title: "Library Reading Section", category: "Academics", url: "https://images.unsplash.com/photo-1521587760476-6c12a4b040da?auto=format&fit=crop&w=600&q=80" },
  { id: "5", title: "Cultural Event Performance", category: "Events", url: "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?auto=format&fit=crop&w=600&q=80" },
  { id: "6", title: "Computer Lab Practice", category: "Academics", url: "https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=600&q=80" },
];

const PAGE_CONTENT = {
  about: {
    en: [
      "Bhagwan Das Inter College serves learners from Ekauna and nearby villages in Chandauli with a focused UP Board education from Classes LKG to 12th.",
      "The institution emphasizes academic rigor, moral values, discipline, and holistic personality development for all students.",
    ],
    hi: [
      "भगवान दास इंटर कॉलेज एकौना और आसपास के गांवों के विद्यार्थियों को एल.के.जी. से 12वीं तक उत्कृष्ट यूपी बोर्ड शिक्षा प्रदान करता है।",
      "संस्थान सभी विद्यार्थियों के लिए शैक्षणिक उत्कृष्टता, नैतिक मूल्यों, अनुशासन और सर्वांगीण व्यक्तित्व विकास पर बल देता है।",
    ],
  },
  academics: {
    en: [
      "Our academic curriculum is aligned with the Uttar Pradesh Secondary Education Board (UPMSP).",
      "Classes LKG through 10th follow compulsory core subjects (Hindi, English, Mathematics, Science, Social Science), while Classes 11th & 12th offer specialized Science, Commerce, and Arts streams.",
    ],
    hi: [
      "हमारा शैक्षणिक पाठ्यक्रम उत्तर प्रदेश माध्यमिक शिक्षा परिषद (यूपी बोर्ड) के नियमों के अनुसार संचालित है।",
      "कक्षा एल.के.जी. से 10वीं तक अनिवार्य मुख्य विषय तथा 11वीं व 12वीं में विज्ञान, वाणिज्य और कला संकाय उपलब्ध हैं।",
    ],
  },
  admissions: {
    en: [
      "Admissions for the current academic session are now open for Classes LKG to 12th grade.",
      "Parents and guardians can apply online or visit the school office. Required documents include Transfer Certificate (TC), Aadhaar card, previous marksheets, and passport-size photographs.",
    ],
    hi: [
      "वर्तमान शैक्षणिक सत्र के लिए कक्षा एल.के.जी. से 12वीं तक प्रवेश प्रक्रिया खुली है।",
      "अभिभावक ऑनलाइन आवेदन कर सकते हैं या विद्यालय कार्यालय संपर्क कर सकते हैं। टी.सी., आधार कार्ड, पिछली अंकपत्र व फोटो आवश्यक हैं।",
    ],
  },
  contact: {
    en: [
      "School Address: Bhagwan Das Inter College, Ekauna, Chandauli, Uttar Pradesh - 232104",
      "Office Hours: Monday to Saturday, 8:00 AM – 2:00 PM",
    ],
    hi: [
      "विद्यालय पता: भगवान दास इंटर कॉलेज, एकौना, चंदौली, उत्तर प्रदेश - 232104",
      "कार्यालय समय: सोमवार से शनिवार, सुबह 8:00 से दोपहर 2:00 बजे तक",
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
  if (key) return dict.nav[key];
  if (first === "privacy") return dict.footer?.privacy || "Privacy Policy";
  if (first === "terms") return dict.footer?.terms || "Terms of Service";
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
      schoolInfo = await fetchPublicSchool({ timeoutMs: 3000 });
    }
  } catch (_err) {
    // Graceful fallback
  }

  const content = PAGE_CONTENT[key];
  const paragraphs = content?.[lang] ?? content?.en ?? [];

  return (
    <div className="mx-auto max-w-5xl px-4 py-12">
      <div className="border-b pb-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold text-stone-900">{heading}</h1>
          <p className="text-xs text-stone-500 mt-1">Bhagwan Das Inter College, Ekauna, Chandauli</p>
        </div>

        {key === "admissions" && (
          <Link
            href={`/${lang}/admissions/apply`}
            className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white rounded-xl text-sm font-semibold hover:bg-emerald-500 shadow-sm transition"
          >
            {lang === "hi" ? "ऑनलाइन प्रवेश फॉर्म भरें →" : "Apply Online Now →"}
          </Link>
        )}
      </div>

      {schoolInfo && (key === "contact" || key === "about") && (
        <div className="card mt-6 p-6 space-y-3 bg-white border border-stone-200 rounded-2xl shadow-sm">
          <p className="font-bold text-lg text-stone-900">
            {lang === "hi" ? schoolInfo.nameHi : schoolInfo.nameEn}
          </p>
          <p className="text-sm text-stone-600">
            📍 {lang === "hi" ? schoolInfo.addressHi : schoolInfo.addressEn}
          </p>
          <p className="text-sm text-stone-600">
            📞 {schoolInfo.phonePrimary || "Office Phone"} {schoolInfo.phoneSecondary ? `/ ${schoolInfo.phoneSecondary}` : ""}
          </p>
          <p className="text-sm text-stone-600">✉️ {schoolInfo.email || "bdicekauna@gmail.com"}</p>
          <p className="text-sm text-stone-600">
            ⏰ {lang === "hi" ? schoolInfo.officeHoursHi : schoolInfo.officeHoursEn}
          </p>
        </div>
      )}

      {/* Gallery Page Special Showcase */}
      {key === "gallery" && (
        <div className="mt-8 space-y-6">
          <div className="flex flex-wrap gap-2">
            <span className="px-3 py-1 bg-emerald-600 text-white text-xs font-semibold rounded-full cursor-pointer">
              {lang === "hi" ? "सभी फ़ोटो" : "All Photos"}
            </span>
            <span className="px-3 py-1 bg-stone-100 text-stone-700 text-xs font-semibold rounded-full hover:bg-stone-200 cursor-pointer">
              {lang === "hi" ? "परिसर" : "Campus"}
            </span>
            <span className="px-3 py-1 bg-stone-100 text-stone-700 text-xs font-semibold rounded-full hover:bg-stone-200 cursor-pointer">
              {lang === "hi" ? "खेलकूद" : "Sports"}
            </span>
            <span className="px-3 py-1 bg-stone-100 text-stone-700 text-xs font-semibold rounded-full hover:bg-stone-200 cursor-pointer">
              {lang === "hi" ? "प्रयोगशालाएं" : "Laboratories"}
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {(dynamicItems && dynamicItems.length > 0 ? dynamicItems : DEMO_GALLERY_IMAGES).map((img, idx) => (
              <div key={img.id || idx} className="group overflow-hidden rounded-2xl border border-stone-200 bg-white shadow-sm hover:shadow-md transition">
                <div className="h-52 bg-stone-100 overflow-hidden relative">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={img.url || img.imageUrl}
                    alt={img.title || "School Gallery"}
                    className="h-full w-full object-cover group-hover:scale-105 transition duration-300"
                  />
                  {img.category && (
                    <span className="absolute top-3 left-3 bg-stone-900/70 text-white text-[10px] font-semibold uppercase px-2.5 py-1 rounded-md backdrop-blur-sm">
                      {img.category}
                    </span>
                  )}
                </div>
                <div className="p-4">
                  <p className="font-semibold text-sm text-stone-900">{img.title || "Campus View"}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Dynamic API items rendering */}
      {key !== "gallery" && dynamicItems && dynamicItems.length > 0 ? (
        <div className="mt-6 space-y-4">
          {dynamicItems.map((item, idx) => (
            <div key={item.id || idx} className="card p-5 border rounded-2xl bg-white shadow-sm border-stone-200">
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
      ) : key !== "gallery" ? (
        <div className="card mt-6 space-y-4 p-6 text-stone-700 bg-white border border-stone-200 rounded-2xl shadow-sm leading-relaxed">
          {paragraphs.map((paragraph, i) => (
            <p key={i} className="text-base text-stone-800">{paragraph}</p>
          ))}
        </div>
      ) : null}

      <p className="mt-10 pt-4 border-t">
        <Link
          href={`/${lang}`}
          className="font-semibold text-emerald-700 underline-offset-2 hover:underline inline-flex items-center gap-1 text-sm"
        >
          ← {dict.nav.home}
        </Link>
      </p>
    </div>
  );
}
