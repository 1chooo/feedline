"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

function HomeIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-6 w-6"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      aria-hidden="true"
    >
      <path d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1z" />
    </svg>
  );
}

export function Sidebar() {
  const pathname = usePathname();
  const home = pathname === "/";

  return (
    <aside className="sticky top-0 hidden h-svh w-20 shrink-0 flex-col items-center px-2 py-6 lg:flex">
      <Link
        href="/"
        className="text-sm font-semibold tracking-tight"
        aria-label="Stream"
      >
        S
      </Link>
      <nav className="mt-8 flex flex-col items-center gap-2">
        <Link
          href="/"
          aria-label="Home"
          className={`rounded-2xl p-3 ${home ? "text-foreground" : "text-muted hover:bg-surface"}`}
        >
          <HomeIcon />
        </Link>
      </nav>
    </aside>
  );
}
