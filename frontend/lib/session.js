"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import * as api from "./api";

/**
 * Who is signed in, held in one place.
 *
 * The token lives in localStorage. That is a deliberate, temporary choice for
 * the local build: the API is on a different port from the site, so an
 * httpOnly cookie would need SameSite=None with Secure, which does not work
 * over plain http on localhost.
 *
 * TODO before deployment: once the API and the site are served from one domain
 * (for example bdic.edu.in and bdic.edu.in/api), move the token into an
 * httpOnly, Secure, SameSite=Lax cookie set by the Go API. Only this file and
 * the API's sign-in handler need to change. Tokens are short-lived (ten hours)
 * and revocable server-side in the meantime.
 */

const STORAGE_KEY = "bdic_session";
const LOCALE_COOKIE = "bdic_locale";

function readStoredSession() {
  // Private browsing, blocked site data, and a first server render all make
  // this throw or return nothing, so every read is guarded.
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw);
    if (!parsed?.accessToken) {
      return null;
    }
    // Drop a token that has already expired rather than sending it and
    // getting a 401 back.
    if (parsed.expiresAt && new Date(parsed.expiresAt) <= new Date()) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function writeStoredSession(session) {
  try {
    if (session) {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Nothing to do: the session simply will not survive a reload.
  }
}

/**
 * Writes a session to storage from outside the provider.
 *
 * The sign-in page is not inside the portal's SessionProvider — it has no
 * chrome and no signed-in user yet — so it stores the token here and then
 * navigates into the portal, where the provider picks it up and revalidates it.
 */
export function persistSession({ accessToken, expiresAt, user }) {
  writeStoredSession({ accessToken, expiresAt, user });
}

/** Reads the stored token, or null. */
export function readStoredToken() {
  return readStoredSession()?.accessToken ?? null;
}

/** Remembers the language choice so proxy.js can honour it on the next visit. */
export function rememberLocale(locale) {
  try {
    const oneYear = 60 * 60 * 24 * 365;
    document.cookie = `${LOCALE_COOKIE}=${locale}; path=/; max-age=${oneYear}; samesite=lax`;
  } catch {
    // Cookies blocked. The language still applies for this page load.
  }
}

const SessionContext = createContext(null);

export function SessionProvider({ children }) {
  const [state, setState] = useState({
    status: "loading", // loading | signedIn | signedOut
    token: null,
    user: null,
    school: null,
    currentSession: null,
  });

  // On first load, revalidate whatever is in storage against the server. A
  // token can be revoked or an account suspended between visits, and the local
  // copy of the user's name and role would then be stale.
  useEffect(() => {
    let cancelled = false;

    const stored = readStoredSession();
    if (!stored) {
      setState({
        status: "signedOut",
        token: null,
        user: null,
        school: null,
        currentSession: null,
      });
      return undefined;
    }

    api
      .fetchMe(stored.accessToken)
      .then((data) => {
        if (cancelled) {
          return;
        }
        setState({
          status: "signedIn",
          token: stored.accessToken,
          user: data.user,
          school: data.school,
          currentSession: data.currentSession,
        });
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        // A network failure is not a reason to throw the session away: the
        // user may simply be in a dead spot. Only an actual rejection is.
        if (error instanceof api.ApiError && error.code === "NETWORK") {
          setState({
            status: "signedIn",
            token: stored.accessToken,
            user: stored.user ?? null,
            school: null,
            currentSession: null,
          });
          return;
        }
        writeStoredSession(null);
        setState({
          status: "signedOut",
          token: null,
          user: null,
          school: null,
          currentSession: null,
        });
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const signIn = useCallback(async (identifier, password) => {
    const result = await api.signIn(identifier, password);
    const session = {
      accessToken: result.accessToken,
      expiresAt: result.expiresAt,
      user: result.user,
    };
    writeStoredSession(session);

    // Pull the school and session in the same breath, so the portal has what
    // it needs without a second round trip on a slow connection.
    let extras = { school: null, currentSession: null };
    try {
      const me = await api.fetchMe(result.accessToken);
      extras = { school: me.school, currentSession: me.currentSession };
    } catch {
      // Not fatal. The portal will fill these in on its next load.
    }

    setState({
      status: "signedIn",
      token: result.accessToken,
      user: result.user,
      ...extras,
    });
    return result.user;
  }, []);

  const signOut = useCallback(async (token) => {
    const current = token ?? readStoredSession()?.accessToken;
    // Tell the server so the sign-out is in the audit trail, but do not let a
    // failed call trap the user in a session they asked to leave.
    if (current) {
      try {
        await api.signOut(current);
      } catch {
        // Ignored on purpose.
      }
    }
    writeStoredSession(null);
    setState({
      status: "signedOut",
      token: null,
      user: null,
      school: null,
      currentSession: null,
    });
  }, []);

  /** Called when any request comes back saying the session is no longer valid. */
  const invalidate = useCallback(() => {
    writeStoredSession(null);
    setState({
      status: "signedOut",
      token: null,
      user: null,
      school: null,
      currentSession: null,
    });
  }, []);

  const setLocale = useCallback(
    async (locale) => {
      rememberLocale(locale);
      const current = state.token;
      if (current) {
        try {
          await api.saveLocale(current, locale);
        } catch {
          // The cookie already carries the choice for this browser.
        }
      }
    },
    [state.token],
  );

  const value = useMemo(
    () => ({ ...state, signIn, signOut, invalidate, setLocale }),
    [state, signIn, signOut, invalidate, setLocale],
  );

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  );
}

export function useSession() {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error("useSession must be used inside a SessionProvider");
  }
  return context;
}
