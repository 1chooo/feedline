import Link from "next/link";
import { ProfileLink } from "@/components/profile-link";
import { SignOutButton } from "@/components/sign-out-button";
import { initials } from "@/lib/initials";
import type { User } from "@/types/social";

export function Header({ user }: { user: User | null }) {
  return (
    <header className="sticky top-0 z-10 border-b border-border bg-surface/95 backdrop-blur lg:hidden">
      <div className="flex h-14 items-center justify-between gap-3 px-4">
        <Link href="/" className="text-base font-semibold tracking-tight">Stream</Link>
        <div className="flex items-center gap-3">
          {user ? <><ProfileLink username={user.username} className="flex h-9 w-9 items-center justify-center rounded-full border border-border text-xs font-medium" aria-label={`Your profile: ${user.displayName}`}>{initials(user.displayName)}</ProfileLink><SignOutButton className="text-sm font-medium" /></> : <Link href="/login" className="rounded-full bg-foreground px-4 py-2 text-sm font-medium text-background">Sign in</Link>}
        </div>
      </div>
      <nav aria-label="Main navigation" className="flex gap-1 overflow-x-auto px-2 pb-2 text-sm">
        <Link href="/" className="shrink-0 rounded-lg px-3 py-2">Home</Link>
        <Link href={user ? "/advertiser" : "/advertise"} className="shrink-0 rounded-lg px-3 py-2">{user ? "Ad workspace" : "Advertise"}</Link>
        {user?.role === "admin" ? <Link href="/admin" className="shrink-0 rounded-lg px-3 py-2">Administration</Link> : null}
        <Link href="/settings" className="shrink-0 rounded-lg px-3 py-2">Settings</Link>
      </nav>
    </header>
  );
}
