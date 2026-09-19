import { notFound } from "next/navigation";
import { Feed } from "@/components/feed";
import { ProfileHeader } from "@/components/profile-header";
import { ApiError, getUser, listPostsByUsername } from "@/lib/api";
import { toFeedItems } from "@/lib/feed";
import type { FeedItem, User } from "@/types/social";

export default async function ProfilePage({
  params,
}: PageProps<"/users/[username]">) {
  const { username } = await params;

  let user: User | null = null;
  let items: FeedItem[] = [];
  let loadError: string | null = null;

  try {
    user = await getUser(username);
    items = toFeedItems(await listPostsByUsername(user.username));
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      notFound();
    }
    loadError = "Can't load this profile. Is the API running?";
  }

  return (
    <main className="flex-1">
      {user ? <ProfileHeader user={user} /> : null}
      {loadError ? (
        <p className="px-4 py-10 text-center text-sm text-muted">{loadError}</p>
      ) : (
        <Feed items={items} />
      )}
    </main>
  );
}
