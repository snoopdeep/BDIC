"use client";

import { useEffect, useId, useRef, useState } from "react";

/**
 * The portal's design system.
 *
 * One file, because these pieces only make sense as a set and a screen should
 * be able to import what it needs in one line. Every one of them is built to
 * the same three rules, which come from who actually uses this:
 *
 *   - 44px minimum tap targets. A teacher marks attendance with a thumb.
 *   - Tables become cards below the `sm` breakpoint. A register with eight
 *     columns is unreadable on a phone, and horizontal scrolling inside a
 *     page is worse.
 *   - No colour without a word beside it. A red dot means nothing to someone
 *     who cannot distinguish it, and nothing at all in a printed register.
 */

/* ------------------------------------------------------------------ *
 * Page furniture
 * ------------------------------------------------------------------ */

export function PageHeader({ title, subtitle, actions }) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div className="min-w-0">
        <h1 className="text-2xl font-bold text-brand-900">{title}</h1>
        {subtitle ? (
          <p className="mt-1 text-sm text-zinc-600">{subtitle}</p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex flex-wrap items-center gap-2">{actions}</div>
      ) : null}
    </div>
  );
}

export function Panel({ title, description, actions, children, className = "" }) {
  return (
    <section className={`card p-4 sm:p-5 ${className}`}>
      {title || actions ? (
        <div className="mb-4 flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0">
            {title ? (
              <h2 className="text-base font-semibold text-brand-900">{title}</h2>
            ) : null}
            {description ? (
              <p className="mt-0.5 text-sm text-zinc-600">{description}</p>
            ) : null}
          </div>
          {actions ? (
            <div className="flex flex-wrap items-center gap-2">{actions}</div>
          ) : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}

/** A single figure on a dashboard. */
export function StatCard({ label, value, detail, tone = "neutral", href }) {
  const tones = {
    neutral: "border-brand-100",
    good: "border-good-600/30 bg-good-50",
    warn: "border-warn-600/30 bg-warn-50",
    bad: "border-bad-600/30 bg-bad-50",
  };

  const content = (
    <>
      <p className="text-xs uppercase tracking-wide text-zinc-500">{label}</p>
      <p className="mt-1 text-2xl font-bold tabular text-brand-900">{value}</p>
      {detail ? <p className="mt-0.5 text-xs text-zinc-600">{detail}</p> : null}
    </>
  );

  const className = `block rounded-card border bg-white p-4 ${tones[tone] ?? tones.neutral}`;

  if (href) {
    return (
      <a href={href} className={`${className} transition hover:border-brand-300`}>
        {content}
      </a>
    );
  }
  return <div className={className}>{content}</div>;
}

/* ------------------------------------------------------------------ *
 * Feedback
 * ------------------------------------------------------------------ */

export function Alert({ tone = "info", children, title }) {
  const tones = {
    info: "bg-brand-50 text-brand-900 border-brand-200",
    good: "bg-good-50 text-good-700 border-good-600/30",
    warn: "bg-warn-50 text-warn-700 border-warn-600/30",
    bad: "bg-bad-50 text-bad-700 border-bad-600/30",
  };

  return (
    <div
      // An error must be announced; a confirmation only needs to be available.
      role={tone === "bad" ? "alert" : "status"}
      className={`rounded-lg border px-3 py-2 text-sm ${tones[tone] ?? tones.info}`}
    >
      {title ? <p className="font-semibold">{title}</p> : null}
      {children}
    </div>
  );
}

export function LoadingBlock({ label = "Loading…", rows = 3 }) {
  return (
    <div aria-busy="true" aria-live="polite">
      <span className="sr-only">{label}</span>
      <div className="space-y-2">
        {Array.from({ length: rows }, (_, index) => (
          <div
            key={index}
            className="h-11 animate-pulse rounded-lg bg-brand-100/70"
          />
        ))}
      </div>
    </div>
  );
}

export function EmptyState({ title, description, action }) {
  return (
    <div className="rounded-lg border border-dashed border-brand-200 bg-brand-50/50 px-4 py-10 text-center">
      <p className="font-medium text-zinc-800">{title}</p>
      {description ? (
        <p className="mx-auto mt-1 max-w-md text-sm text-zinc-600">{description}</p>
      ) : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}

/** A status word with a colour, never a colour on its own. */
export function Badge({ tone = "neutral", children }) {
  const tones = {
    neutral: "bg-zinc-100 text-zinc-700",
    brand: "bg-brand-100 text-brand-800",
    good: "bg-good-100 text-good-700",
    warn: "bg-warn-100 text-warn-700",
    bad: "bg-bad-100 text-bad-700",
  };

  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${tones[tone] ?? tones.neutral}`}
    >
      {children}
    </span>
  );
}

/* ------------------------------------------------------------------ *
 * Controls
 * ------------------------------------------------------------------ */

export function Button({
  children,
  variant = "primary",
  size = "md",
  busy = false,
  type = "button",
  className = "",
  ...rest
}) {
  const variants = {
    primary: "bg-brand-700 text-white hover:bg-brand-800",
    secondary: "border border-brand-200 bg-white text-brand-800 hover:bg-brand-50",
    quiet: "text-brand-700 hover:bg-brand-50",
    danger: "bg-bad-600 text-white hover:bg-bad-700",
  };
  const sizes = {
    // Never below 44px: this is operated with a thumb.
    md: "min-h-11 px-4 text-sm",
    lg: "min-h-12 px-5 text-base",
  };

  return (
    <button
      type={type}
      disabled={busy || rest.disabled}
      aria-busy={busy || undefined}
      className={`inline-flex items-center justify-center gap-2 rounded-lg font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${variants[variant] ?? variants.primary} ${sizes[size] ?? sizes.md} ${className}`}
      {...rest}
    >
      {busy ? (
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          className="size-4 animate-spin"
          fill="none"
        >
          <circle
            cx="12"
            cy="12"
            r="9"
            stroke="currentColor"
            strokeWidth="3"
            opacity="0.25"
          />
          <path
            d="M21 12a9 9 0 0 0-9-9"
            stroke="currentColor"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
      ) : null}
      {children}
    </button>
  );
}

const controlClass =
  "block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2.5 text-base outline-hidden transition focus:border-brand-600 focus:ring-2 focus:ring-brand-200 disabled:bg-zinc-50 disabled:text-zinc-500";

/**
 * A labelled control with its own error message.
 *
 * `error` comes straight from the API's per-field messages, so the box that is
 * wrong is the box that says so — rather than a banner at the top of a long
 * form and the reader hunting for which field it meant.
 */
export function Field({
  label,
  hint,
  error,
  required = false,
  children,
  htmlFor,
}) {
  const generated = useId();
  const id = htmlFor ?? generated;
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;

  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-zinc-800">
        {label}
        {required ? (
          <span aria-hidden="true" className="ms-0.5 text-bad-600">
            *
          </span>
        ) : null}
      </label>
      <div className="mt-1">
        {typeof children === "function"
          ? children({
              id,
              "aria-invalid": error ? true : undefined,
              "aria-describedby": errorId ?? hintId,
              className: controlClass,
            })
          : children}
      </div>
      {error ? (
        <p id={errorId} className="mt-1 text-xs text-bad-700">
          {error}
        </p>
      ) : hint ? (
        <p id={hintId} className="mt-1 text-xs text-zinc-500">
          {hint}
        </p>
      ) : null}
    </div>
  );
}

export function TextInput({ className = "", ...rest }) {
  return <input className={`${controlClass} ${className}`} {...rest} />;
}

export function TextArea({ className = "", rows = 4, ...rest }) {
  return (
    <textarea rows={rows} className={`${controlClass} ${className}`} {...rest} />
  );
}

export function Select({ children, className = "", ...rest }) {
  return (
    <select className={`${controlClass} ${className}`} {...rest}>
      {children}
    </select>
  );
}

export function Checkbox({ label, id: providedId, ...rest }) {
  const generated = useId();
  const id = providedId ?? generated;
  return (
    <label
      htmlFor={id}
      className="flex min-h-11 cursor-pointer items-center gap-3 text-sm text-zinc-800"
    >
      <input
        id={id}
        type="checkbox"
        className="size-5 rounded border-zinc-300 text-brand-700 focus:ring-2 focus:ring-brand-200"
        {...rest}
      />
      <span>{label}</span>
    </label>
  );
}

/* ------------------------------------------------------------------ *
 * Tables
 * ------------------------------------------------------------------ */

/**
 * A table that turns into a list of cards on a phone.
 *
 * `columns` is [{ key, header, render?, align?, hideOnMobile?, isPrimary? }].
 * The column marked `isPrimary` becomes each card's heading; anything marked
 * `hideOnMobile` is left out of the card entirely, which is how an eight-column
 * register stays readable on a 375px screen.
 */
export function DataTable({
  columns,
  rows,
  rowKey,
  empty,
  caption,
  onRowClick,
  footer,
}) {
  if (!rows || rows.length === 0) {
    return empty ?? null;
  }

  const keyFor = rowKey ?? ((row, index) => row.id ?? index);
  const primary = columns.find((column) => column.isPrimary) ?? columns[0];
  const secondary = columns.filter(
    (column) => column !== primary && !column.hideOnMobile,
  );

  function cellValue(column, row) {
    return column.render ? column.render(row) : row[column.key];
  }

  const alignClass = (column) =>
    column.align === "right"
      ? "text-right"
      : column.align === "center"
        ? "text-center"
        : "text-left";

  return (
    <>
      {/* ---- Phone: one card per row --------------------------------- */}
      <ul className="space-y-2 sm:hidden">
        {rows.map((row, index) => (
          <li key={keyFor(row, index)}>
            <div
              className={`card p-3 ${onRowClick ? "cursor-pointer active:bg-brand-50" : ""}`}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
              role={onRowClick ? "button" : undefined}
              tabIndex={onRowClick ? 0 : undefined}
              onKeyDown={
                onRowClick
                  ? (event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        onRowClick(row);
                      }
                    }
                  : undefined
              }
            >
              <p className="font-semibold text-brand-900">
                {cellValue(primary, row)}
              </p>
              <dl className="mt-2 grid grid-cols-2 gap-x-3 gap-y-1.5">
                {secondary.map((column) => (
                  <div key={column.key} className="min-w-0">
                    <dt className="text-[11px] uppercase tracking-wide text-zinc-500">
                      {column.header}
                    </dt>
                    <dd className="truncate text-sm text-zinc-800">
                      {cellValue(column, row)}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          </li>
        ))}
      </ul>

      {/* ---- Everything else: a real table --------------------------- */}
      <div className="hidden overflow-x-auto sm:block">
        <table className="w-full border-collapse text-sm">
          {caption ? <caption className="sr-only">{caption}</caption> : null}
          <thead>
            <tr className="border-b border-brand-100">
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  className={`px-3 py-2 text-xs font-semibold uppercase tracking-wide text-zinc-500 ${alignClass(column)}`}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr
                key={keyFor(row, index)}
                className={`border-b border-brand-50 ${onRowClick ? "cursor-pointer hover:bg-brand-50" : ""}`}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={`px-3 py-2.5 align-middle text-zinc-800 ${alignClass(column)} ${column.numeric ? "tabular" : ""}`}
                  >
                    {cellValue(column, row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
          {footer ? <tfoot>{footer}</tfoot> : null}
        </table>
      </div>
    </>
  );
}

export function Pagination({ total, limit, offset, onChange, labels }) {
  if (total <= limit) {
    return null;
  }

  const page = Math.floor(offset / limit) + 1;
  const pages = Math.ceil(total / limit);

  return (
    <nav
      className="mt-4 flex items-center justify-between gap-3"
      aria-label={labels?.pagination ?? "Pages"}
    >
      <Button
        variant="secondary"
        disabled={offset === 0}
        onClick={() => onChange(Math.max(0, offset - limit))}
      >
        {labels?.previous ?? "Previous"}
      </Button>
      <p className="text-sm text-zinc-600">
        {page} / {pages}
      </p>
      <Button
        variant="secondary"
        disabled={offset + limit >= total}
        onClick={() => onChange(offset + limit)}
      >
        {labels?.next ?? "Next"}
      </Button>
    </nav>
  );
}

/* ------------------------------------------------------------------ *
 * Modal
 * ------------------------------------------------------------------ */

/**
 * A dialog that behaves like one: Escape closes it, focus moves into it and
 * returns when it closes, and the page behind does not scroll.
 */
export function Modal({ open, onClose, title, children, footer, wide = false }) {
  const panelRef = useRef(null);
  const openerRef = useRef(null);

  useEffect(() => {
    if (!open) {
      return undefined;
    }

    openerRef.current = document.activeElement;

    function onKeyDown(event) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    panelRef.current?.focus();

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previousOverflow;
      if (openerRef.current instanceof HTMLElement) {
        openerRef.current.focus();
      }
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center">
      <button
        type="button"
        aria-label="Close"
        onClick={onClose}
        className="absolute inset-0 bg-zinc-900/50"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className={`relative max-h-[92dvh] w-full overflow-y-auto rounded-t-2xl bg-white p-5 outline-hidden sm:rounded-2xl ${wide ? "sm:max-w-3xl" : "sm:max-w-lg"}`}
      >
        <h2 className="text-lg font-semibold text-brand-900">{title}</h2>
        <div className="mt-4">{children}</div>
        {footer ? (
          <div className="mt-6 flex flex-wrap justify-end gap-2">{footer}</div>
        ) : null}
      </div>
    </div>
  );
}

/**
 * Asks before doing something that cannot be undone.
 *
 * Used for cancelling a receipt, publishing a result, approving a photograph —
 * the places where a mis-tap has consequences outside the screen.
 */
export function ConfirmButton({
  onConfirm,
  title,
  message,
  confirmLabel,
  cancelLabel,
  children,
  variant = "danger",
  busy = false,
  requireReason = false,
  reasonLabel,
}) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");

  return (
    <>
      <Button variant={variant} onClick={() => setOpen(true)} busy={busy}>
        {children}
      </Button>

      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title={title}
        footer={
          <>
            <Button variant="secondary" onClick={() => setOpen(false)}>
              {cancelLabel}
            </Button>
            <Button
              variant={variant}
              disabled={requireReason && reason.trim().length < 3}
              onClick={() => {
                setOpen(false);
                onConfirm(requireReason ? reason.trim() : undefined);
                setReason("");
              }}
            >
              {confirmLabel}
            </Button>
          </>
        }
      >
        <p className="text-sm text-zinc-700">{message}</p>
        {requireReason ? (
          <div className="mt-4">
            <Field label={reasonLabel} required>
              {(props) => (
                <TextArea
                  {...props}
                  rows={3}
                  value={reason}
                  onChange={(event) => setReason(event.target.value)}
                />
              )}
            </Field>
          </div>
        ) : null}
      </Modal>
    </>
  );
}

/* ------------------------------------------------------------------ *
 * Filters
 * ------------------------------------------------------------------ */

/** The row of dropdowns above a list. Wraps instead of scrolling on a phone. */
export function FilterBar({ children }) {
  return (
    <div className="mb-4 flex flex-wrap items-end gap-3">{children}</div>
  );
}

/** A labelled dropdown sized for a filter bar rather than a form. */
export function FilterSelect({ label, value, onChange, children }) {
  const id = useId();
  return (
    <div className="min-w-40">
      <label htmlFor={id} className="block text-xs font-medium text-zinc-600">
        {label}
      </label>
      <Select
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 py-2"
      >
        {children}
      </Select>
    </div>
  );
}

/** A search box that reports what was typed, debounced by the caller. */
export function SearchInput({ value, onChange, placeholder, label }) {
  const id = useId();
  return (
    <div className="min-w-52 flex-1">
      <label htmlFor={id} className="block text-xs font-medium text-zinc-600">
        {label}
      </label>
      <TextInput
        id={id}
        type="search"
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 py-2"
      />
    </div>
  );
}

/** Wraps a block that should be printable on its own, such as a receipt. */
export function Printable({ children }) {
  return <div className="print:m-0 print:p-0">{children}</div>;
}
