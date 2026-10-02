"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import * as api from "./api";
import { useSession } from "./session";

/**
 * Data fetching for the portal.
 *
 * Every screen needs the same four things: the token attached, a loading
 * state, an error in the reader's language, and a way to reload after a save.
 * Doing that once here is why the screens themselves are short.
 */

/**
 * Loads data from the API as soon as a token is available.
 *
 * @param {(token: string, signal: AbortSignal) => Promise<any>} loader
 * @param {object} [options]
 * @param {any[]} [options.deps]  Re-runs when these change, like useEffect.
 * @param {boolean} [options.skip] Do not load at all (a filter is not chosen yet).
 * @returns {{data, error, loading, reload, setData}}
 */
export function useApiData(loader, options = {}) {
  const { deps = [], skip = false } = options;
  const { token, invalidate } = useSession();

  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(!skip);
  const [reloadCount, setReloadCount] = useState(0);

  // Held in a ref so changing the loader's identity on every render — which a
  // arrow function passed inline does — does not restart the request.
  const loaderRef = useRef(loader);
  loaderRef.current = loader;

  useEffect(() => {
    if (skip || !token) {
      setLoading(false);
      return undefined;
    }

    const controller = new AbortController();
    let cancelled = false;

    setLoading(true);
    setError(null);

    loaderRef
      .current(token, controller.signal)
      .then((result) => {
        if (!cancelled) {
          setData(result);
          setError(null);
        }
      })
      .catch((caught) => {
        if (cancelled || controller.signal.aborted) {
          return;
        }
        // A revoked token means the session is over. Clearing it sends the
        // guard to the sign-in page instead of showing a wall of errors.
        if (caught instanceof api.ApiError && caught.needsSignIn) {
          invalidate();
          return;
        }
        setError(caught);
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
      controller.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, skip, reloadCount, invalidate, ...deps]);

  const reload = useCallback(() => {
    setReloadCount((count) => count + 1);
  }, []);

  return { data, error, loading, reload, setData };
}

/**
 * Runs a write and reports its progress.
 *
 * @param {(token: string, input: any) => Promise<any>} action
 * @returns {{run, busy, error, fieldErrors, reset, done}}
 */
export function useApiAction(action) {
  const { token, invalidate } = useSession();

  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);
  const [fieldErrors, setFieldErrors] = useState({});
  const [done, setDone] = useState(false);

  const actionRef = useRef(action);
  actionRef.current = action;

  const reset = useCallback(() => {
    setError(null);
    setFieldErrors({});
    setDone(false);
  }, []);

  const run = useCallback(
    async (input) => {
      if (!token) {
        return { ok: false };
      }

      setBusy(true);
      setError(null);
      setFieldErrors({});
      setDone(false);

      try {
        const result = await actionRef.current(token, input);
        setDone(true);
        return { ok: true, result };
      } catch (caught) {
        if (caught instanceof api.ApiError) {
          if (caught.needsSignIn) {
            invalidate();
            return { ok: false };
          }
          setFieldErrors(caught.fields || {});
          setError(caught);
        } else {
          setError(caught);
        }
        return { ok: false, error: caught };
      } finally {
        setBusy(false);
      }
    },
    [token, invalidate],
  );

  return { run, busy, error, fieldErrors, reset, done };
}

/**
 * Turns an error into a sentence in the reader's language.
 *
 * The server sends validation text in both languages, so that is used when
 * present. The dictionary covers the codes that never reach the server, such
 * as a connection that failed.
 */
export function errorMessage(error, dict, locale) {
  if (!error) {
    return null;
  }
  if (error instanceof api.ApiError) {
    return (
      error.message_for(locale) ||
      dict?.errors?.[error.code] ||
      dict?.errors?.INTERNAL ||
      "Something went wrong."
    );
  }
  return dict?.common?.somethingWentWrong || "Something went wrong.";
}

/**
 * A debounced value, for search boxes.
 *
 * Without this, typing a student's name fires one request per keystroke — on a
 * 4G connection in a village that is a queue of requests the office watches
 * resolve out of order.
 */
export function useDebounced(value, delay = 350) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}

/** The current academic session id, for screens that filter by it. */
export function useCurrentSessionId() {
  const { currentSession } = useSession();
  return currentSession?.id ?? null;
}
