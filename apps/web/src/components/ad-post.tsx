import { ProfileLink } from "@/components/profile-link";
import Image from "next/image";
import { initials } from "@/lib/initials";
import { adClickUrl } from "@/lib/api";
import type { FeedItem } from "@/types/social";

export function AdPost({ item }: { item: FeedItem }) {
  const { post, author, kind } = item;

  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-surface">
      {kind === "ad" ? (
        <div className="flex items-center gap-3 px-4 py-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-full border border-border bg-background text-xs font-medium">
            Ad
          </span>
          <span className="min-w-0">
            <span className="block truncate font-medium">Sponsored</span>
            <span className="block truncate text-sm text-muted">
              Promoted on Stream
            </span>
          </span>
        </div>
      ) : author ? (
        <ProfileLink
          username={author.username}
          className="flex items-center gap-3 px-4 py-3"
        >
          <span className="flex h-10 w-10 items-center justify-center rounded-full border border-border bg-background text-sm font-medium">
            {initials(author.displayName)}
          </span>
          <span className="min-w-0">
            <span className="block truncate font-medium">{author.displayName}</span>
            <span className="block truncate text-sm text-muted">
              @{author.username}
            </span>
          </span>
        </ProfileLink>
      ) : null}
      {post.imageUrl ? (
        <Image
          src={post.imageUrl}
          alt={post.title}
          width={800}
          height={800}
          unoptimized
          className="aspect-square w-full object-cover"
        />
      ) : null}
      <div className="space-y-2 px-4 py-3">
        <h2 className="text-base font-semibold">{post.title}</h2>
        {post.description ? (
          <p className="text-sm leading-6 text-muted">{post.description}</p>
        ) : null}
        {post.landingPageUrl ? (
          <a
            href={kind === "ad" ? adClickUrl(post.id) : post.landingPageUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-block text-sm font-medium underline-offset-2 hover:underline"
          >
            Visit
          </a>
        ) : null}
      </div>
    </article>
  );
}
