import Link from "next/link";
import { ProfileLink } from "@/components/profile-link";
import { SignOutButton } from "@/components/sign-out-button";
import { initials } from "@/lib/initials";
import type { User } from "@/types/social";

export function AccountPanel({ user }: { user: User }) {
  return (
    <aside className="sticky top-0 h-svh w-full max-w-80 px-6 py-10">
      <p className="text-sm text-muted">Signed in</p>
      <ProfileLink username={user.username} className="mt-4 flex items-center gap-3">
        <span className="flex h-12 w-12 items-center justify-center rounded-full border border-border bg-surface text-sm font-medium">
          {initials(user.displayName)}
        </span>
        <span className="min-w-0">
          <span className="block truncate font-medium">{user.displayName}</span>
          <span className="block truncate text-sm text-muted">@{user.username}</span>
        </span>
      </ProfileLink>
      {user.bio ? (
        <p className="mt-4 text-sm leading-6 text-muted">{user.bio}</p>
      ) : null}
      <Link
        href="/advertiser"
        className="mt-6 block w-full rounded-full bg-foreground py-2.5 text-center text-sm font-medium text-background"
      >
        {user.role === "advertiser" || user.role === "admin"
          ? "Ad dashboard"
          : "Start advertising"}
      </Link>
      <SignOutButton className="mt-3 w-full rounded-full border border-border py-2.5 text-sm font-medium" />
    </aside>
  );
}
