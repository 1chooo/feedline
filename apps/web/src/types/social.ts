export type User = {
  username: string;
  displayName: string;
  bio: string;
};

export type Post = {
  id: string;
  username: string;
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
  endAt?: string;
};
