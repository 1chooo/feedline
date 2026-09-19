import { Feed } from "@/components/feed";
import { getAllPosts, getUser } from "@/data/mock";

export default function Home() {
  const items = getAllPosts().flatMap((post) => {
    const author = getUser(post.username);
    return author ? [{ post, author }] : [];
  });

  return (
    <main className="flex-1">
      <h1 className="hidden border-b border-border px-4 py-3 text-base font-semibold lg:block">
        Home
      </h1>
      <Feed items={items} />
    </main>
  );
}
