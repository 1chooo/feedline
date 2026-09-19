"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const items = [
  { href: "/", label: "For you" },
  { href: "/settings", label: "Settings" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="sticky top-0 hidden h-svh w-56 shrink-0 flex-col px-3 py-6 lg:flex">
      <Link href="/" className="px-3 text-lg font-semibold tracking-tight">
        Stream
      </Link>
      <nav className="mt-6 flex flex-col gap-1">
        {items.map((item) => {
          const active = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`rounded-xl px-3 py-2 text-base ${
                active ? "font-semibold" : "text-muted hover:bg-surface"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <button
        type="button"
        className="mt-6 rounded-full bg-foreground px-3.5 py-2 text-sm font-medium text-background"
      >
        Sign in
      </button>
    </aside>
  );
}
