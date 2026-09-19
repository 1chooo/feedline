import { ComposeForm } from "@/components/compose-form";
import { Feed } from "@/components/feed";
import { getCurrentUser, listAds, listPosts } from "@/lib/api";
import { interleaveAds, toFeedItems } from "@/lib/feed";

export default async function Home() {
  const user = await getCurrentUser();

  let items = toFeedItems([]);
  let loadError: string | null = null;

  try {
    const posts = toFeedItems(await listPosts());
    const ads = await listAds(
      user
        ? { age: user.age, gender: user.gender, country: user.country }
        : undefined,
    );
    items = interleaveAds(posts, ads);
  } catch {
    loadError = "Can't load the feed. Is the API running?";
  }

  return (
    <main className="flex-1">
      <h1 className="hidden border-b border-border px-4 py-3 text-base font-semibold lg:block">
        Home
      </h1>
      {user ? <ComposeForm /> : null}
      {loadError ? (
        <p className="px-4 py-10 text-center text-sm text-muted">{loadError}</p>
      ) : (
        <Feed items={items} />
      )}
    </main>
  );
}
