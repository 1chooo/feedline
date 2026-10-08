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

function ProfileIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-6 w-6"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      aria-hidden="true"
    >
      <circle cx="12" cy="8" r="3" />
      <path d="M5 19c1.4-3 4-4.5 7-4.5S17.6 16 19 19" />
    </svg>
  );
}

function SettingsIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-6 w-6"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="3" />
      <path d="M12 4.5v1.5M12 18v1.5M4.5 12H6M18 12h1.5M6.8 6.8l1.1 1.1M16.1 16.1l1.1 1.1M6.8 17.2l1.1-1.1M16.1 7.9l1.1-1.1" />
    </svg>
  );
}

function CampaignIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-6 w-6"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      aria-hidden="true"
    >
      <path d="M5 20V5m0-5v10h4l7 3V2l-7 3H5" />
      <path d="M9 15v4" />
    </svg>
  );
}

function NavLink({
  href,
  label,
  active,
  children,
}: {
  href: string;
  label: string;
  active: boolean;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      aria-label={label}
      className={`rounded-2xl p-3 ${active ? "text-foreground" : "text-muted hover:bg-surface"}`}
    >
      {children}
    </Link>
  );
}

export function Sidebar({ username }: { username?: string }) {
  const pathname = usePathname();

  return (
    <aside className="sticky top-0 hidden h-svh w-20 shrink-0 flex-col items-center px-2 py-6 lg:flex">
      <Link
        href="/"
        className="text-sm font-semibold tracking-tight"
        aria-label="Stream"
      >
        S
      </Link>
      <nav className="mt-8 flex flex-1 flex-col items-center gap-2">
        <NavLink href="/" label="Home" active={pathname === "/"}>
          <HomeIcon />
        </NavLink>
        {username ? (
          <>
            <NavLink
              href={`/@${username}`}
              label="Profile"
              active={
                pathname === `/users/${username}` || pathname === `/@${username}`
              }
            >
              <ProfileIcon />
            </NavLink>
            <NavLink
              href="/advertiser"
              label="Advertise"
              active={pathname === "/advertiser"}
            >
              <CampaignIcon />
            </NavLink>
          </>
        ) : null}
      </nav>
      <NavLink
        href="/settings"
        label="Settings"
        active={pathname === "/settings"}
      >
        <SettingsIcon />
      </NavLink>
    </aside>
  );
}
