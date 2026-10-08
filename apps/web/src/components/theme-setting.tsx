"use client";

import { useSyncExternalStore } from "react";

function subscribe(onChange: () => void) {
  window.addEventListener("stream-theme", onChange);
  window.addEventListener("storage", onChange);
  return () => {
    window.removeEventListener("stream-theme", onChange);
    window.removeEventListener("storage", onChange);
  };
}

export function ThemeSetting() {
  const dark = useSyncExternalStore(subscribe, () => document.documentElement.classList.contains("dark"), () => false);
  function setTheme(nextDark: boolean) {
    document.documentElement.classList.toggle("dark", nextDark);
    localStorage.setItem("theme", nextDark ? "dark" : "light");
    window.dispatchEvent(new Event("stream-theme"));
  }
  return <div role="group" aria-label="Color theme" className="grid grid-cols-2 gap-2">{[false, true].map((value) => <button key={String(value)} type="button" aria-pressed={dark === value} onClick={() => setTheme(value)} className={`rounded-xl border px-3 py-2 text-sm font-medium ${dark === value ? "border-foreground text-foreground" : "border-border text-muted"}`}>{value ? "Dark" : "Light"}</button>)}</div>;
}
