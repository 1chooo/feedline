import { ComposeForm } from "@/components/compose-form";
import { Feed } from "@/components/feed";
import { getCurrentUser, listAds, listPosts } from "@/lib/api";
import { loadFeed } from "@/lib/feed";

export default async function Home() {
  const user = await getCurrentUser();

  const { items, error: loadError } = await loadFeed(
    listPosts,
    () => listAds(
      user
        ? { age: user.age, gender: user.gender, country: user.country }
        : undefined,
    ),
  );

  return (
    <main className="flex-1">
      <h1 className="border-b border-border px-4 py-3 text-base font-semibold">
        Home
      </h1>
      {user ? <ComposeForm /> : null}
      {loadError ? (
        <p role="alert" className="px-4 py-10 text-center text-sm text-muted">{loadError}</p>
      ) : (
        <Feed items={items} />
      )}
    </main>
  );
}
