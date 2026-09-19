import { initials } from "@/lib/initials";
import type { User } from "@/types/social";

export function ProfileHeader({ user }: { user: User }) {
  return (
    <div className="mx-auto flex w-full max-w-lg items-start gap-4 px-4 pt-6 pb-2">
      <span className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full border border-border bg-surface text-lg font-medium">
        {initials(user.displayName)}
      </span>
      <div className="min-w-0 flex-1">
        <h1 className="text-xl font-semibold tracking-tight">
          {user.displayName}
        </h1>
        <p className="text-sm text-muted">@{user.username}</p>
        <p className="mt-2 text-sm leading-6 text-muted">{user.bio}</p>
      </div>
    </div>
  );
}
