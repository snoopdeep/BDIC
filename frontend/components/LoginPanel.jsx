"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import * as api from "../lib/api";
import { persistSession } from "../lib/session";

/**
 * One field's validation message, from the server.
 *
 * The API returns a per-field message alongside the banner text, and without
 * something like this it is collected and thrown away — leaving the user with
 * "please check what you have entered" and no idea which box is wrong.
 */
function FieldError({ id, message }) {
  if (!message) {
    return null;
  }
  return (
    <p id={id} className="mt-1 text-xs text-bad-700">
      {message}
    </p>
  );
}

/**
 * Sign in, and the two steps of a password reset.
 *
 * One identifier field, because a teacher thinks of an email, a parent thinks
 * of a phone number, and a student thinks of an admission number. Making them
 * choose the right kind first is a step that only ever goes wrong.
 */
export default function LoginPanel({ lang, dict }) {
  const router = useRouter();
  const [mode, setMode] = useState("signIn"); // signIn | forgot | reset
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [fieldErrors, setFieldErrors] = useState({});

  function handleFailure(caught) {
    if (caught instanceof api.ApiError) {
      setFieldErrors(caught.fields || {});
      // The server sends the message in both languages, so a validation
      // message is already in the reader's. The dictionary covers the codes
      // that never reach the server, such as a network failure.
      setError(
        caught.message_for(lang) || dict.errors[caught.code] || dict.errors.INTERNAL,
      );
      return;
    }
    setError(dict.common.somethingWentWrong);
  }

  async function onSignIn(event) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setNotice(null);
    setFieldErrors({});

    try {
      const result = await api.signIn(identifier.trim(), password);
      persistSession(result);
      // If this account must change its password, the portal shows a
      // persistent banner. Setting a notice here would be invisible: the
      // redirect below unmounts this panel immediately.
      router.replace(`/${lang}/portal`);
    } catch (caught) {
      handleFailure(caught);
    } finally {
      setBusy(false);
    }
  }

  async function onRequestCode(event) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setFieldErrors({});

    try {
      await api.requestPasswordReset(identifier.trim());
      // Deliberately the same answer whether or not the account exists, so
      // this page cannot be used to find out whose phone number is registered.
      setNotice(dict.login.codeSent);
      setMode("reset");
    } catch (caught) {
      handleFailure(caught);
    } finally {
      setBusy(false);
    }
  }

  async function onReset(event) {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    if (newPassword !== confirmPassword) {
      setError(dict.login.passwordsDoNotMatch);
      setFieldErrors({ confirmPassword: dict.login.passwordsDoNotMatch });
      return;
    }

    setBusy(true);
    try {
      await api.resetPassword(identifier.trim(), code.trim(), newPassword);
      setNotice(dict.login.resetDone);
      setMode("signIn");
      setPassword("");
      setCode("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (caught) {
      handleFailure(caught);
    } finally {
      setBusy(false);
    }
  }

  const inputClass =
    "mt-1 block w-full rounded-lg border border-zinc-300 px-3 py-3 text-base outline-hidden transition focus:border-brand-600 focus:ring-2 focus:ring-brand-200";
  const labelClass = "block text-sm font-medium text-zinc-800";
  const buttonClass =
    "mt-6 inline-flex min-h-12 w-full items-center justify-center rounded-lg bg-brand-700 px-4 text-base font-semibold text-white transition hover:bg-brand-800 disabled:opacity-60";

  return (
    <div className="w-full max-w-md">
      <div className="card p-6 sm:p-8">
        <h1 className="text-2xl font-bold text-brand-900">
          {mode === "reset" ? dict.login.resetTitle : dict.login.title}
        </h1>
        <p className="mt-1 text-sm text-zinc-600">
          {mode === "forgot"
            ? dict.login.forgotHelp
            : mode === "reset"
              ? dict.login.forgotHelp
              : dict.login.subtitle}
        </p>

        {notice ? (
          <p
            role="status"
            className="mt-4 rounded-lg bg-good-50 px-3 py-2 text-sm text-good-700"
          >
            {notice}
          </p>
        ) : null}

        {error ? (
          <p
            role="alert"
            className="mt-4 rounded-lg bg-bad-50 px-3 py-2 text-sm text-bad-700"
          >
            {error}
          </p>
        ) : null}

        {/* ---- Sign in -------------------------------------------- */}
        {mode === "signIn" ? (
          <form onSubmit={onSignIn} className="mt-6" noValidate>
            <div>
              <label htmlFor="identifier" className={labelClass}>
                {dict.login.identifier}
              </label>
              <input
                id="identifier"
                name="identifier"
                type="text"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                required
                autoComplete="username"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck="false"
                aria-invalid={Boolean(fieldErrors.identifier)}
                aria-describedby={
                  fieldErrors.identifier ? "identifier-error" : "identifier-help"
                }
                className={inputClass}
              />
              <FieldError id="identifier-error" message={fieldErrors.identifier} />
              <p id="identifier-help" className="mt-1 text-xs text-zinc-500">
                {dict.login.identifierHelp}
              </p>
            </div>

            <div className="mt-4">
              <label htmlFor="password" className={labelClass}>
                {dict.login.password}
              </label>
              <div className="relative">
                <input
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                  autoComplete="current-password"
                  aria-invalid={Boolean(fieldErrors.password)}
                  className={`${inputClass} pe-24`}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((value) => !value)}
                  className="absolute inset-y-0 end-0 px-3 text-sm font-medium text-brand-700"
                >
                  {showPassword
                    ? dict.login.hidePassword
                    : dict.login.showPassword}
                </button>
              </div>
              <FieldError id="password-error" message={fieldErrors.password} />
            </div>

            <button type="submit" disabled={busy} className={buttonClass}>
              {busy ? dict.login.submitting : dict.login.submit}
            </button>

            <button
              type="button"
              onClick={() => {
                setMode("forgot");
                setError(null);
                setNotice(null);
              }}
              className="mt-4 w-full text-sm font-medium text-brand-700 underline-offset-2 hover:underline"
            >
              {dict.login.forgotPassword}
            </button>
          </form>
        ) : null}

        {/* ---- Ask for a code ------------------------------------- */}
        {mode === "forgot" ? (
          <form onSubmit={onRequestCode} className="mt-6" noValidate>
            <div>
              <label htmlFor="forgot-identifier" className={labelClass}>
                {dict.login.identifier}
              </label>
              <input
                id="forgot-identifier"
                type="text"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                required
                autoComplete="username"
                autoCapitalize="none"
                className={inputClass}
              />
            </div>

            <button type="submit" disabled={busy} className={buttonClass}>
              {busy ? dict.common.saving : dict.login.sendCode}
            </button>

            <button
              type="button"
              onClick={() => {
                setMode("signIn");
                setError(null);
              }}
              className="mt-4 w-full text-sm font-medium text-brand-700 underline-offset-2 hover:underline"
            >
              {dict.login.backToLogin}
            </button>
          </form>
        ) : null}

        {/* ---- Use the code --------------------------------------- */}
        {mode === "reset" ? (
          <form onSubmit={onReset} className="mt-6" noValidate>
            <div>
              <label htmlFor="reset-identifier" className={labelClass}>
                {dict.login.identifier}
              </label>
              <input
                id="reset-identifier"
                type="text"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                required
                autoComplete="username"
                autoCapitalize="none"
                className={inputClass}
              />
            </div>

            <div className="mt-4">
              <label htmlFor="code" className={labelClass}>
                {dict.login.code}
              </label>
              <input
                id="code"
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                maxLength={6}
                value={code}
                onChange={(event) => setCode(event.target.value)}
                required
                autoComplete="one-time-code"
                aria-invalid={Boolean(fieldErrors.code)}
                className={`${inputClass} tabular tracking-widest`}
              />
              <FieldError id="code-error" message={fieldErrors.code} />
            </div>

            <div className="mt-4">
              <label htmlFor="new-password" className={labelClass}>
                {dict.login.newPassword}
              </label>
              <input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                required
                autoComplete="new-password"
                aria-invalid={Boolean(fieldErrors.newPassword || fieldErrors.password)}
                aria-describedby={
                  fieldErrors.newPassword || fieldErrors.password
                    ? "new-password-error"
                    : undefined
                }
                className={inputClass}
              />
              <FieldError
                id="new-password-error"
                message={fieldErrors.newPassword || fieldErrors.password}
              />
            </div>

            <div className="mt-4">
              <label htmlFor="confirm-password" className={labelClass}>
                {dict.login.confirmPassword}
              </label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                required
                autoComplete="new-password"
                aria-invalid={Boolean(fieldErrors.confirmPassword)}
                aria-describedby={
                  fieldErrors.confirmPassword ? "confirm-password-error" : undefined
                }
                className={inputClass}
              />
              <FieldError
                id="confirm-password-error"
                message={fieldErrors.confirmPassword}
              />
            </div>

            <button type="submit" disabled={busy} className={buttonClass}>
              {busy ? dict.common.saving : dict.login.resetSubmit}
            </button>

            <button
              type="button"
              onClick={() => {
                setMode("signIn");
                setError(null);
              }}
              className="mt-4 w-full text-sm font-medium text-brand-700 underline-offset-2 hover:underline"
            >
              {dict.login.backToLogin}
            </button>
          </form>
        ) : null}
      </div>

      <p className="mt-6 text-center text-sm text-zinc-600">
        {dict.login.needHelp}
      </p>
      <p className="mt-2 text-center text-sm">
        <Link
          href={`/${lang}`}
          className="font-medium text-brand-700 underline-offset-2 hover:underline"
        >
          {dict.login.backToSite}
        </Link>
      </p>
    </div>
  );
}
