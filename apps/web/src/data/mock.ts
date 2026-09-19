import type { Post, User } from "@/types/social";

export const users: User[] = [
  {
    username: "jane",
    displayName: "Jane Park",
    bio: "Northline Outdoor. Gear for slow weekends outside.",
  },
  {
    username: "kai",
    displayName: "Kai Rivera",
    bio: "Harbor Roast. Small-batch coffee from the waterfront.",
  },
  {
    username: "nova",
    displayName: "Nova Chen",
    bio: "Lumen Labs. Quiet tech for busy days.",
  },
  {
    username: "miles",
    displayName: "Miles Ortega",
    bio: "Notes on walking, cities, and ordinary ads that feel human.",
  },
];

export const posts: Post[] = [
  {
    id: "1",
    username: "jane",
    title: "The pack that disappears on the trail",
    description: "12 liters, no bounce, and a bottle pocket you can reach with one hand.",
    imageUrl: "https://picsum.photos/id/1015/800/800",
    landingPageUrl: "https://example.com/northline-pack",
    endAt: "2026-12-31T00:00:00.000Z",
  },
  {
    id: "2",
    username: "kai",
    title: "Harbor blend, roasted this morning",
    description: "Chocolate, orange peel, and a finish that stays with the walk home.",
    imageUrl: "https://picsum.photos/id/1080/800/800",
    landingPageUrl: "https://example.com/harbor-roast",
    endAt: "2026-11-15T00:00:00.000Z",
  },
  {
    id: "3",
    username: "nova",
    title: "Headphones that stay out of the way",
    description: "Soft clamp, 30-hour battery, and a case that fits a jacket pocket.",
    imageUrl: "https://picsum.photos/id/180/800/800",
    landingPageUrl: "https://example.com/lumen-quiet",
    endAt: "2026-10-01T00:00:00.000Z",
  },
  {
    id: "4",
    username: "miles",
    title: "Why I still walk to work",
    description:
      "Twenty minutes each way. No playlist. Just the same block of shops and one billboard that finally learned my name.",
    landingPageUrl: "https://example.com/miles-walk",
    endAt: "2026-09-30T00:00:00.000Z",
  },
  {
    id: "5",
    username: "jane",
    title: "Weekend tent for two",
    description: "Sets up in the time it takes to boil water. Packs smaller than a loaf of bread.",
    imageUrl: "https://picsum.photos/id/1016/800/800",
    landingPageUrl: "https://example.com/northline-tent",
    endAt: "2026-12-31T00:00:00.000Z",
  },
];

function normalizeUsername(username: string) {
  return username.replace(/^@/, "").toLowerCase();
}

export function getUser(username: string) {
  const key = normalizeUsername(username);
  return users.find((user) => user.username === key);
}

export function getAllPosts() {
  return posts;
}

export function getPostsByUsername(username: string) {
  const key = normalizeUsername(username);
  return posts.filter((post) => post.username === key);
}
