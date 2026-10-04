export type User = {
  username: string;
  displayName: string;
  bio: string;
  age?: number;
  gender?: string;
  country?: string;
};

export type Post = {
  id: string;
  username: string;
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
  createdAt?: string;
  endAt?: string;
};

export type FeedKind = "post" | "ad";

export type FeedItem = {
  kind: FeedKind;
  post: Post;
  author?: User;
};
