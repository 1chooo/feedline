import type { FeedItem, Post, User } from "@/types/social";

type AdListItem = {
  id: number;
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
  endAt: string;
};

export function toFeedItems(items: { post: Post; author: User }[]): FeedItem[] {
  return items.map((item) => ({
    kind: "post",
    post: item.post,
    author: item.author,
  }));
}

export function interleaveAds(
  posts: FeedItem[],
  ads: AdListItem[],
): FeedItem[] {
  const result: FeedItem[] = [];
  let adIndex = 0;

  if (posts.length === 0) {
    return ads.map((ad) => adItem(ad));
  }

  posts.forEach((item, index) => {
    result.push(item);
    const shouldInsert = adIndex < ads.length && (index === 0 || (index + 1) % 2 === 0);
    if (shouldInsert) {
      result.push(adItem(ads[adIndex]));
      adIndex += 1;
    }
  });

  while (adIndex < ads.length) {
    result.push(adItem(ads[adIndex]));
    adIndex += 1;
  }

  return result;
}

function adItem(ad: AdListItem): FeedItem {
  return {
    kind: "ad",
    post: {
      id: String(ad.id),
      username: "sponsored",
      title: ad.title,
      description: ad.description,
      imageUrl: ad.imageUrl,
      landingPageUrl: ad.landingPageUrl,
      endAt: ad.endAt,
    },
  };
}
