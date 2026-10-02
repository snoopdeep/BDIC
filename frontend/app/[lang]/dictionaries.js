import "server-only";

/**
 * Translations.
 *
 * Every string the interface shows is written out in both languages here, in
 * advance. Nothing is machine-translated at the moment of viewing, because
 * machine-translated Hindi reads like translated English and a parent notices.
 *
 * One file per module rather than one enormous file, so two people working on
 * two modules never edit the same JSON. They are merged at the top level: a
 * module file contributes its own key (`admissions`, `fees`, …) into the
 * dictionary the screens read.
 *
 * Adding a module:
 *   1. Create dictionaries/<module>.en.json and <module>.hi.json, each a
 *      single top-level object keyed by the module name.
 *   2. Add the module to MODULES below.
 * Nothing else changes.
 */

const MODULES = [
  "core",
  "site",
  "admissions",
  "people",
  "academics",
  "teaching",
  "exams",
  "fees",
  "comms",
  "dashboard",
];

const loaders = {
  en: {
    core: () => import("./dictionaries/en.json"),
    site: () => import("./dictionaries/site.en.json"),
    admissions: () => import("./dictionaries/admissions.en.json"),
    people: () => import("./dictionaries/people.en.json"),
    academics: () => import("./dictionaries/academics.en.json"),
    teaching: () => import("./dictionaries/teaching.en.json"),
    exams: () => import("./dictionaries/exams.en.json"),
    fees: () => import("./dictionaries/fees.en.json"),
    comms: () => import("./dictionaries/comms.en.json"),
    dashboard: () => import("./dictionaries/dashboard.en.json"),
  },
  hi: {
    core: () => import("./dictionaries/hi.json"),
    site: () => import("./dictionaries/site.hi.json"),
    admissions: () => import("./dictionaries/admissions.hi.json"),
    people: () => import("./dictionaries/people.hi.json"),
    academics: () => import("./dictionaries/academics.hi.json"),
    teaching: () => import("./dictionaries/teaching.hi.json"),
    exams: () => import("./dictionaries/exams.hi.json"),
    fees: () => import("./dictionaries/fees.hi.json"),
    comms: () => import("./dictionaries/comms.hi.json"),
    dashboard: () => import("./dictionaries/dashboard.hi.json"),
  },
};

export const LOCALES = ["hi", "en"];
export const DEFAULT_LOCALE = "hi";

export function hasLocale(locale) {
  return Object.hasOwn(loaders, locale);
}

/**
 * Loads and merges the dictionary for one language.
 *
 * Cached per locale for the life of the process: these are static JSON files,
 * and re-reading ten of them on every render of every page would be waste.
 */
const cache = new Map();

export async function getDictionary(locale) {
  const key = hasLocale(locale) ? locale : DEFAULT_LOCALE;

  const cached = cache.get(key);
  if (cached) {
    return cached;
  }

  const moduleLoaders = loaders[key];
  const parts = await Promise.all(
    MODULES.map(async (name) => {
      const loaded = await moduleLoaders[name]();
      return loaded.default;
    }),
  );

  // A shallow merge at the top level is deliberate. Each module owns its own
  // key, so there is nothing to merge deeply — and a deep merge would let one
  // module silently overwrite half of another's strings.
  const merged = Object.assign({}, ...parts);
  cache.set(key, merged);
  return merged;
}
