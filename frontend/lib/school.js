import { cache } from "react";

import { fetchPublicSchool } from "./api";

/**
 * The school's details for a Server Component, with a fallback.
 *
 * The public pages must still render if the API is not running — which is the
 * normal state five seconds after someone clones this repository and starts
 * only the frontend. A page that explodes with a fetch error in that situation
 * teaches the reader nothing; one that renders with the school's name and a
 * quiet note does.
 */

const FALLBACK = {
  nameEn: "Bhagwan Das Inter College",
  nameHi: "भगवान दास इंटर कॉलेज",
  shortName: "BDIC",
  board: "UP Board (UPMSP)",
  affiliationNo: "",
  udiseCode: "",
  addressEn: "Ekauna, Chandauli, Uttar Pradesh",
  addressHi: "एकौना, चंदौली, उत्तर प्रदेश",
  village: "Ekauna",
  district: "Chandauli",
  state: "Uttar Pradesh",
  pincode: "",
  phonePrimary: "",
  phoneSecondary: "",
  email: "",
  officeHoursEn: "Monday to Saturday, 8:00 AM to 2:00 PM",
  officeHoursHi: "सोमवार से शनिवार, सुबह 8:00 से दोपहर 2:00 बजे तक",
  principalName: "",
  mapEmbedUrl: "",
  instagramUrl: "https://www.instagram.com/bdic1146/",
  facebookUrl: "https://www.facebook.com/groups/1206203054416295/",
  googlePlaceUrl: "",
  academicYear: "",
};

/**
 * Wrapped in React's cache() so one page render makes one request.
 *
 * Rendering the home page asks for the school in four places — the layout's
 * metadata, the layout body, the site chrome, and the page itself. Without
 * this they would be four separate round trips, and four separate four-second
 * timeouts when the API is down.
 *
 * @returns {Promise<{school: object, apiReachable: boolean}>}
 */
export const getSchool = cache(async () => {
  try {
    const school = await fetchPublicSchool({ timeoutMs: 4000 });
    return { school: { ...FALLBACK, ...school }, apiReachable: true };
  } catch {
    return { school: FALLBACK, apiReachable: false };
  }
});

/** Picks the right language's field, falling back to the other one. */
export function pick(record, field, locale) {
  const suffix = locale === "hi" ? "Hi" : "En";
  const other = locale === "hi" ? "En" : "Hi";
  return record?.[`${field}${suffix}`] || record?.[`${field}${other}`] || "";
}
