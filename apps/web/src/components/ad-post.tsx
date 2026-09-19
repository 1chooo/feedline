import { ProfileLink } from "@/components/profile-link";
import type { Post, User } from "@/types/social";

function initials(name: string) {
  return name
    .split(" ")
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase();
}

export function AdPost({ post, author }: { post: Post; author: User }) {
  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-surface">
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
      {post.imageUrl ? (
        <img
          src={post.imageUrl}
          alt={post.title}
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
            href={post.landingPageUrl}
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
