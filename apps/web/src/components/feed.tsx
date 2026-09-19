import { AdPost } from "@/components/ad-post";
import type { FeedItem } from "@/types/social";

export function Feed({ items }: { items: FeedItem[] }) {
  if (items.length === 0) {
    return (
      <p className="px-4 py-10 text-center text-sm text-muted">No posts yet.</p>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-lg flex-col gap-4 px-4 py-4">
      {items.map((item) => (
        <AdPost key={`${item.kind}-${item.post.id}`} item={item} />
      ))}
    </div>
  );
}
