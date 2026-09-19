import { Feed } from "@/components/feed";
import { getAllPosts, getUser } from "@/data/mock";

export default function Home() {
  const items = getAllPosts().flatMap((post) => {
    const author = getUser(post.username);
    return author ? [{ post, author }] : [];
  });

  return (
    <main className="flex-1">
      <Feed items={items} />
    </main>
  );
}
