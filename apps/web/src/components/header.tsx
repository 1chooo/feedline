import Link from "next/link";
import { ProfileLink } from "@/components/profile-link";
import { SignOutButton } from "@/components/sign-out-button";
import { initials } from "@/lib/initials";
import type { User } from "@/types/social";

export function Header({ user }: { user: User | null }) {
  return (
    <header className="sticky top-0 z-10 border-b border-border bg-surface/90 backdrop-blur lg:hidden">
      <div className="mx-auto flex h-14 w-full max-w-lg items-center justify-between px-4">
        <Link href="/" className="text-base font-semibold tracking-tight">
          Stream
        </Link>
        <div className="flex items-center gap-3">
          <Link
            href="/settings"
            className="text-sm font-medium underline-offset-2 hover:underline"
          >
            Settings
          </Link>
          {user ? (
            <>
              <Link
                href="/advertiser"
                className="text-sm font-medium underline-offset-2 hover:underline"
              >
                Advertise
              </Link>
              <ProfileLink
                username={user.username}
                className="flex h-8 w-8 items-center justify-center rounded-full border border-border text-xs font-medium"
                aria-label={user.displayName}
              >
                {initials(user.displayName)}
              </ProfileLink>
              <SignOutButton className="text-sm font-medium underline-offset-2 hover:underline" />
            </>
          ) : (
            <Link
              href="/login"
              className="rounded-full bg-foreground px-3.5 py-1.5 text-sm font-medium text-background"
            >
              Sign in
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
