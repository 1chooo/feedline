"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { User } from "@/types/social";

export function Sidebar({ user }: { user: User | null }) {
  const pathname = usePathname();
  const links = [
    { href: "/", label: "Home" },
    { href: "/advertise", label: "Advertising services" },
    ...(user ? [
      { href: `/users/${user.username}`, label: "Your profile" },
      { href: "/advertiser", label: user.role === "member" ? "Start advertising" : "Ad workspace" },
    ] : []),
    ...(user?.role === "admin" ? [{ href: "/admin", label: "Administration" }] : []),
    { href: "/settings", label: "Settings" },
  ];
  return (
    <aside className="sticky top-0 hidden h-svh w-full flex-col px-4 py-6 lg:flex">
      <Link href="/" className="px-3 text-xl font-semibold tracking-tight">Stream</Link>
      <nav aria-label="Main navigation" className="mt-8 flex flex-col gap-2">
        {links.map(({ href, label }) => {
          const active = pathname === href;
          return <Link key={href} href={href} aria-current={active ? "page" : undefined} className={`rounded-xl px-3 py-3 text-sm font-medium ${active ? "bg-surface text-foreground" : "text-muted hover:bg-surface"}`}>{label}</Link>;
        })}
      </nav>
      <p className="mt-auto px-3 text-xs leading-5 text-muted">A place to share.<br />A space to grow.</p>
    </aside>
  );
}
