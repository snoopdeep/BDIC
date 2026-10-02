/**
 * The one place that talks to the Go API.
 *
 * Every response the API can return is turned into either data or an ApiError
 * carrying a machine code and both languages, so no screen has to interpret an
 * HTTP status or a raw fetch failure for itself.
 */

export const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/**
 * An error from the API, or from failing to reach it.
 *
 * `code` is what the interface switches on. `messageEn` and `messageHi` come
 * from the server so a validation message is already in the user's language;
 * the dictionary's errors block is the fallback for codes we invent here.
 */
export class ApiError extends Error {
  constructor({ status, code, messageEn, messageHi, fields, cause }) {
    super(messageEn || code || "Request failed", { cause });
    this.name = "ApiError";
    this.status = status ?? 0;
    this.code = code || "INTERNAL";
    this.messageEn = messageEn || "";
    this.messageHi = messageHi || "";
    this.fields = fields || {};
  }

  /** The message in the reader's language, falling back to the other one. */
  message_for(locale) {
    if (locale === "hi") {
      return this.messageHi || this.messageEn;
    }
    return this.messageEn || this.messageHi;
  }

  /** True when signing in again is the fix. */
  get needsSignIn() {
    return this.code === "UNAUTHORIZED" || this.code === "TOKEN_REVOKED";
  }
}

/**
 * Perform a request.
 *
 * @param {string} path      Path beginning with "/", e.g. "/api/v1/me".
 * @param {object} [options]
 * @param {string} [options.method]
 * @param {any}    [options.body]   Serialised as JSON when present.
 * @param {string} [options.token]  Bearer token for authenticated calls.
 * @param {AbortSignal} [options.signal]
 * @param {number} [options.timeoutMs]
 * @param {RequestCache} [options.cache]
 * @param {number} [options.revalidate]  Seconds. Public reads only.
 * @param {string[]} [options.tags]      Cache tags, for targeted revalidation.
 */
export async function request(path, options = {}) {
  const {
    method = "GET",
    body,
    token,
    signal,
    timeoutMs = 20000,
    cache,
    revalidate,
    tags,
  } = options;

  const headers = { Accept: "application/json" };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  // A request that never returns would leave a spinner on screen forever,
  // which on a patchy village connection is the normal case, not the edge one.
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  if (signal) {
    signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  /*
    Caching is opt-in in this version of Next, so the default has to be chosen
    deliberately per request rather than left to the framework.

    Anything carrying a bearer token, and anything that is not a GET, is
    never cached: one family's data must never be reused for another, and a
    cached POST is a bug by definition.

    A public GET — the school's address, the class list — is the same for
    everybody, so it is cached. That is also what lets the public pages
    prerender at build time instead of blocking on the API for every visitor.
  */
  const isPrivate = Boolean(token) || method !== "GET";
  const cacheMode = cache ?? (isPrivate ? "no-store" : "force-cache");
  const nextOptions =
    !isPrivate && (revalidate !== undefined || tags)
      ? { revalidate, tags }
      : undefined;

  let response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
      cache: cacheMode,
      ...(nextOptions ? { next: nextOptions } : {}),
    });
  } catch (cause) {
    // Keep the original error as the cause. Without it, a genuine bug — a bad
    // URL, a TLS failure — is indistinguishable in the logs from a parent
    // standing in a dead spot.
    throw new ApiError({
      status: 0,
      code: "NETWORK",
      messageEn:
        "Could not reach the school server. Check your connection and try again.",
      messageHi:
        "विद्यालय सर्वर से संपर्क नहीं हो सका। कृपया इंटरनेट जाँचकर दोबारा कोशिश करें।",
      cause,
    });
  } finally {
    clearTimeout(timer);
  }

  if (response.status === 204) {
    return null;
  }

  const raw = await response.text();
  let payload = null;
  if (raw) {
    try {
      payload = JSON.parse(raw);
    } catch {
      payload = null;
    }
  }

  if (!response.ok) {
    const envelope = payload?.error ?? {};
    throw new ApiError({
      status: response.status,
      code: envelope.code,
      messageEn: envelope.messageEn,
      messageHi: envelope.messageHi,
      fields: envelope.fields,
    });
  }

  return payload;
}

/* ------------------------------------------------------------------ *
 * Public endpoints. No token, so these are safe to call from a Server
 * Component while rendering the website.
 * ------------------------------------------------------------------ */

/*
  The school's own details change perhaps twice a year, so five minutes of
  cache is generous and still means the public pages are not re-fetching this
  for every visitor. The tag lets a future School-settings save revalidate it
  at once rather than waiting the five minutes out.
*/
export function fetchPublicSchool(options) {
  return request("/api/v1/public/school", {
    revalidate: 300,
    tags: ["school"],
    ...options,
  });
}

export function fetchPublicStructure(options) {
	return request("/api/v1/public/structure", {
		revalidate: 300,
		tags: ["structure"],
		...options,
	});
}

// Public editorial content changes more often than the school profile. A
// short revalidation window makes the preview feel current without making a
// request for every visitor.
export function fetchPublicNotices(options) {
	return request("/api/v1/public/site/notices", {
		revalidate: 60,
		tags: ["public-notices"],
		...options,
	});
}

export function fetchPublicEvents(options) {
	return request("/api/v1/public/site/events", {
		revalidate: 60,
		tags: ["public-events"],
		...options,
	});
}

export function fetchPublicAchievements(options) {
	return request("/api/v1/public/site/achievements", {
		revalidate: 300,
		tags: ["public-achievements"],
		...options,
	});
}

export function fetchPublicFaculty(options) {
	return request("/api/v1/public/site/faculty", {
		revalidate: 300,
		tags: ["public-faculty"],
		...options,
	});
}

export function fetchPublicFacilities(options) {
	return request("/api/v1/public/site/facilities", {
		revalidate: 300,
		tags: ["public-facilities"],
		...options,
	});
}

/* ------------------------------------------------------------------ *
 * Authentication
 * ------------------------------------------------------------------ */

export function signIn(identifier, password) {
  return request("/api/v1/auth/login", {
    method: "POST",
    body: { identifier, password },
  });
}

export function signOut(token) {
  return request("/api/v1/auth/logout", { method: "POST", token });
}

export function requestPasswordReset(identifier) {
  return request("/api/v1/auth/forgot-password", {
    method: "POST",
    body: { identifier },
  });
}

export function resetPassword(identifier, code, newPassword) {
  return request("/api/v1/auth/reset-password", {
    method: "POST",
    body: { identifier, code, newPassword },
  });
}

export function changePassword(token, currentPassword, newPassword) {
  return request("/api/v1/me/password", {
    method: "POST",
    token,
    body: { currentPassword, newPassword },
  });
}

/* ------------------------------------------------------------------ *
 * Signed-in endpoints
 * ------------------------------------------------------------------ */

export function fetchMe(token, options) {
  return request("/api/v1/me", { ...options, token });
}

export function saveLocale(token, locale) {
  return request("/api/v1/me/locale", {
    method: "POST",
    token,
    body: { locale },
  });
}

export function fetchClasses(token, options) {
  return request("/api/v1/classes", { ...options, token });
}

export function fetchSubjects(token, options) {
  return request("/api/v1/subjects", { ...options, token });
}

export function fetchSessions(token, options) {
  return request("/api/v1/sessions", { ...options, token });
}
