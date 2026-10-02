"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";

/**
 * The navigation for a phone.
 *
 * Most visitors to this site are on a phone, so the mobile menu is not an
 * afterthought: it closes on navigation, closes on Escape, traps nothing, and
 * every row is a full-width 48px tap target.
 */
export default function MobileNav({ items, lang, labels }) {
  const [isOpen, setIsOpen] = useState(false);
  const pathname = usePathname();
  const panelRef = useRef(null);
  const toggleRef = useRef(null);

  // Close when the route changes, so tapping a link does not leave the menu
  // sitting open over the page the reader asked for.
  useEffect(() => {
    setIsOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!isOpen) {
      return undefined;
    }

    function onKeyDown(event) {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    }

    document.addEventListener("keydown", onKeyDown);
    // Stop the page behind scrolling while the panel is over it.
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    // Move focus into the panel. Without this a keyboard or screen-reader
    // user opens the menu and stays behind it, reading the page they were
    // trying to navigate away from.
    panelRef.current?.focus();

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = previousOverflow;
      // And put it back where it came from.
      toggleRef.current?.focus();
    };
  }, [isOpen]);

  return (
    <>
      <button
        ref={toggleRef}
        type="button"
        onClick={() => setIsOpen(true)}
        aria-label={labels.openMenu}
        aria-expanded={isOpen}
        aria-controls="mobile-nav-panel"
        className="grid size-11 place-items-center rounded-lg border border-brand-200 text-brand-800 lg:hidden"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          className="size-5"
        >
          <path d="M4 7h16M4 12h16M4 17h16" strokeLinecap="round" />
        </svg>
      </button>

      {isOpen ? (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button
            type="button"
            aria-label={labels.closeMenu}
            onClick={() => setIsOpen(false)}
            className="absolute inset-0 bg-zinc-900/50"
          />
          <div
            id="mobile-nav-panel"
            ref={panelRef}
            role="dialog"
            aria-modal="true"
            aria-label={labels.menu}
            // -1 so it can be focused programmatically without becoming a tab
            // stop of its own.
            tabIndex={-1}
            className="absolute inset-y-0 right-0 flex w-[85%] max-w-sm flex-col bg-white shadow-xl outline-hidden"
          >
            <div className="flex items-center justify-between border-b border-brand-100 px-4 py-3">
              <span className="font-semibold text-brand-900">
                {labels.menu}
              </span>
              <button
                type="button"
                onClick={() => setIsOpen(false)}
                aria-label={labels.closeMenu}
                className="grid size-11 place-items-center rounded-lg text-zinc-600 hover:bg-zinc-100"
              >
                <svg
                  aria-hidden="true"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  className="size-5"
                >
                  <path d="M6 6l12 12M18 6L6 18" strokeLinecap="round" />
                </svg>
              </button>
            </div>

            <nav className="flex-1 overflow-y-auto p-2">
              {items.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="flex min-h-12 items-center rounded-lg px-3 text-base font-medium text-zinc-800 hover:bg-brand-50"
                >
                  {item.label}
                </Link>
              ))}
            </nav>

            <div className="border-t border-brand-100 p-4">
              <Link
                href={`/${lang}/login`}
                className="flex min-h-12 items-center justify-center rounded-lg bg-brand-700 px-4 font-semibold text-white"
              >
                {labels.portalLogin}
              </Link>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
