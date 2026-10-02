/**
 * Formatting, done the Indian way.
 *
 * A rupee amount is ₹1,25,000 — two digits per group after the first three —
 * not ₹125,000. A date is 15 August 2026, not 8/15/2026. Getting either wrong
 * makes the whole system read as foreign software, and worse, makes a fee
 * figure easy to misread.
 */

/** Money is stored and sent as integer paise. Never as a float. */
export function rupeesFromPaise(paise) {
  return (Number(paise) || 0) / 100;
}

/**
 * Formats paise as rupees.
 *
 * @param {number} paise
 * @param {object} [options]
 * @param {boolean} [options.withSymbol=true]
 * @param {boolean} [options.paiseIfPresent=true] Show decimals only when the
 *   amount is not a whole number of rupees, so a fee list reads ₹1,800 rather
 *   than ₹1,800.00 and still shows ₹1,800.50 when that is the real figure.
 */
export function formatMoney(paise, options = {}) {
  const { withSymbol = true, paiseIfPresent = true } = options;
  const amount = rupeesFromPaise(paise);
  const hasPaise = Number(paise) % 100 !== 0;
  const decimals = paiseIfPresent && hasPaise ? 2 : 0;

  const formatter = new Intl.NumberFormat("en-IN", {
    style: withSymbol ? "currency" : "decimal",
    currency: "INR",
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });

  return formatter.format(amount);
}

/** Parses a rupee figure a person typed into integer paise. */
export function paiseFromInput(value) {
  const cleaned = String(value ?? "")
    .replace(/[₹,\s]/g, "")
    .trim();
  if (cleaned === "") {
    return null;
  }
  const amount = Number(cleaned);
  if (!Number.isFinite(amount) || amount < 0) {
    return null;
  }
  // Round rather than truncate, so 10.005 does not quietly lose half a paisa.
  return Math.round(amount * 100);
}

const MONTHS = {
  en: [
    "January", "February", "March", "April", "May", "June",
    "July", "August", "September", "October", "November", "December",
  ],
  hi: [
    "जनवरी", "फ़रवरी", "मार्च", "अप्रैल", "मई", "जून",
    "जुलाई", "अगस्त", "सितंबर", "अक्तूबर", "नवंबर", "दिसंबर",
  ],
};

const WEEKDAYS = {
  en: ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"],
  hi: ["रविवार", "सोमवार", "मंगलवार", "बुधवार", "गुरुवार", "शुक्रवार", "शनिवार"],
};

/**
 * Formats an ISO date (YYYY-MM-DD) for reading.
 *
 * The string is split rather than passed through `new Date()`, because
 * `new Date("2026-08-15")` is parsed as UTC midnight and then rendered in the
 * viewer's time zone — which west of Greenwich shows the day before. An
 * attendance date must never shift.
 */
export function formatDate(isoDate, locale = "hi", options = {}) {
  const { short = false, withWeekday = false } = options;
  if (!isoDate || typeof isoDate !== "string") {
    return "";
  }

  const [year, month, day] = isoDate.slice(0, 10).split("-").map(Number);
  if (!year || !month || !day) {
    return isoDate;
  }

  const monthNames = MONTHS[locale] ?? MONTHS.hi;
  const monthName = monthNames[month - 1] ?? String(month);
  const trimmedMonth = short ? monthName.slice(0, 3) : monthName;

  let text = `${day} ${trimmedMonth} ${year}`;

  if (withWeekday) {
    // Constructed in local time from the parts, so the weekday matches the
    // date as written.
    const weekdayNames = WEEKDAYS[locale] ?? WEEKDAYS.hi;
    const weekday = weekdayNames[new Date(year, month - 1, day).getDay()];
    text = `${weekday}, ${text}`;
  }

  return text;
}

/** Formats a timestamp for an activity log or a receipt. */
export function formatDateTime(isoString, locale = "hi") {
  if (!isoString) {
    return "";
  }
  const when = new Date(isoString);
  if (Number.isNaN(when.getTime())) {
    return isoString;
  }

  const date = formatDate(isoString.slice(0, 10), locale, { short: true });
  const time = new Intl.DateTimeFormat(locale === "hi" ? "hi-IN" : "en-IN", {
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
    timeZone: "Asia/Kolkata",
  }).format(when);

  return `${date}, ${time}`;
}

/** Formats a 24-hour HH:MM from the database as a readable clock time. */
export function formatTime(value, locale = "hi") {
  if (!value) {
    return "";
  }
  const [hours, minutes] = String(value).slice(0, 5).split(":").map(Number);
  if (Number.isNaN(hours)) {
    return value;
  }
  const suffix = hours < 12 ? (locale === "hi" ? "पूर्वाह्न" : "AM") : locale === "hi" ? "अपराह्न" : "PM";
  const display = hours % 12 === 0 ? 12 : hours % 12;
  return `${display}:${String(minutes || 0).padStart(2, "0")} ${suffix}`;
}

/** Today, in the school's time zone, as YYYY-MM-DD. */
export function todayISO() {
  // en-CA gives YYYY-MM-DD, and the explicit time zone means "today" is today
  // in Chandauli rather than wherever the device thinks it is.
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Kolkata",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

/** A whole-number percentage, for attendance and results. */
export function formatPercent(value, locale = "hi", decimals = 0) {
  if (value === null || value === undefined || value === "") {
    return "—";
  }
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return "—";
  }
  return new Intl.NumberFormat(locale === "hi" ? "hi-IN" : "en-IN", {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(number) + "%";
}

/** A plain number with Indian digit grouping. */
export function formatNumber(value, locale = "hi") {
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return "—";
  }
  return new Intl.NumberFormat(locale === "hi" ? "hi-IN" : "en-IN").format(number);
}

/** Picks the field in the reader's language, falling back to the other. */
export function localised(record, field, locale) {
  if (!record) {
    return "";
  }
  const own = locale === "hi" ? `${field}Hi` : `${field}En`;
  const other = locale === "hi" ? `${field}En` : `${field}Hi`;
  return record[own] || record[other] || "";
}

/**
 * How many days until a date, or since it. Negative means overdue.
 *
 * Compared as plain calendar days in the school's time zone, so "due today"
 * means today in Chandauli.
 */
export function daysUntil(isoDate) {
  if (!isoDate) {
    return null;
  }
  const today = todayISO();
  const a = Date.parse(`${today}T00:00:00Z`);
  const b = Date.parse(`${isoDate.slice(0, 10)}T00:00:00Z`);
  if (Number.isNaN(a) || Number.isNaN(b)) {
    return null;
  }
  return Math.round((b - a) / 86400000);
}
