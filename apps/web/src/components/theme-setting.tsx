"use client";

import { useEffect, useState } from "react";

function isDark() {
  return document.documentElement.classList.contains("dark");
}

export function ThemeSetting() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    setDark(isDark());
  }, []);

  function setTheme(nextDark: boolean) {
    document.documentElement.classList.toggle("dark", nextDark);
    localStorage.setItem("theme", nextDark ? "dark" : "light");
    setDark(nextDark);
  }

  return (
    <div className="grid grid-cols-2 gap-2">
      <button
        type="button"
        onClick={() => setTheme(false)}
        className={`rounded-xl border px-3 py-2 text-sm font-medium ${
          dark
            ? "border-border text-muted"
            : "border-foreground text-foreground"
        }`}
      >
        Light
      </button>
      <button
        type="button"
        onClick={() => setTheme(true)}
        className={`rounded-xl border px-3 py-2 text-sm font-medium ${
          dark
            ? "border-foreground text-foreground"
            : "border-border text-muted"
        }`}
      >
        Dark
      </button>
    </div>
  );
}
