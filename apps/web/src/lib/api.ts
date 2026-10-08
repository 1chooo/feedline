import { cookies } from "next/headers";
import type { Post, User } from "@/types/social";

export const SESSION_COOKIE = "session";
const SESSION_MAX_AGE = 60 * 60 * 24 * 30;

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type PostResponse = Post & {
  author: User;
};

type ListResponse<T> = {
  items: T[];
};

type AuthResponse = {
  token: string;
  user: User;
};

type AdListItem = {
  id: number;
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
  endAt: string;
};

export type Media = {
  id: number;
  url: string;
  contentType: string;
  sizeBytes: number;
  createdAt: string;
};

export type AdvertiserAd = {
  id: number;
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
  bid?: number;
  dailyBudget?: number;
  status: "active" | "paused" | "archived";
  startAt: string;
  endAt: string;
  createdAt: string;
};

export type AnalyticsSummary = {
  from: string;
  to: string;
  impressions: number;
  clicks: number;
  ctr: number;
  daily: Array<{ date: string; impressions: number; clicks: number }>;
};

function apiUrl() {
  return process.env.API_URL || "http://localhost:8080";
}

async function request<T>(
  path: string,
  init: RequestInit & { token?: string | null } = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (init.token) {
    headers.set("Authorization", `Bearer ${init.token}`);
  }

  const response = await fetch(`${apiUrl()}${path}`, {
    ...init,
    headers,
    cache: "no-store",
    signal: init.signal ?? AbortSignal.timeout(4000),
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  const data = text ? JSON.parse(text) : null;

  if (!response.ok) {
    throw new ApiError(
      response.status,
      data?.error?.code ?? "ERROR",
      data?.error?.message ?? response.statusText,
    );
  }

  return data as T;
}

export async function getSessionToken() {
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value ?? null;
}

export async function setSessionCookie(token: string) {
  const store = await cookies();
  store.set(SESSION_COOKIE, token, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: SESSION_MAX_AGE,
    secure: process.env.NODE_ENV === "production",
  });
}

export async function clearSessionCookie() {
  const store = await cookies();
  store.delete(SESSION_COOKIE);
}

export async function getCurrentUser(): Promise<User | null> {
  const token = await getSessionToken();
  if (!token) {
    return null;
  }

  try {
    return await request<User>("/api/v1/me", { token });
  } catch {
    return null;
  }
}

export async function login(input: {
  email?: string;
  username?: string;
  password: string;
}) {
  return request<AuthResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function register(input: {
  username: string;
  email: string;
  password: string;
  displayName: string;
  bio?: string;
}) {
  return request<AuthResponse>("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function logout() {
  const token = await getSessionToken();
  if (!token) {
    return;
  }
  try {
    await request<void>("/api/v1/auth/logout", { method: "POST", token });
  } catch {
    // Session is cleared locally even if the API is unreachable.
  }
}

export async function getUser(username: string) {
  return request<User>(`/api/v1/users/${encodeURIComponent(username)}`);
}

function toPost(item: PostResponse): { post: Post; author: User } {
  return {
    post: {
      id: item.id,
      username: item.author.username,
      title: item.title,
      description: item.description,
      imageUrl: item.imageUrl,
      landingPageUrl: item.landingPageUrl,
      createdAt: item.createdAt,
    },
    author: item.author,
  };
}

export async function listPosts() {
  const data = await request<ListResponse<PostResponse>>("/api/v1/posts");
  return data.items.map(toPost);
}

export async function listPostsByUsername(username: string) {
  const data = await request<ListResponse<PostResponse>>(
    `/api/v1/users/${encodeURIComponent(username)}/posts`,
  );
  return data.items.map(toPost);
}

export async function createPost(input: {
  title: string;
  description?: string;
  imageUrl?: string;
  landingPageUrl?: string;
}) {
  const token = await getSessionToken();
  const item = await request<PostResponse>("/api/v1/posts", {
    method: "POST",
    token,
    body: JSON.stringify(input),
  });
  return toPost(item);
}

export async function listAds(profile?: {
  age?: number;
  gender?: string;
  country?: string;
}) {
  const params = new URLSearchParams({ platform: "web", limit: "3" });
  if (profile?.age) {
    params.set("age", String(profile.age));
  }
  if (profile?.gender) {
    params.set("gender", profile.gender);
  }
  if (profile?.country) {
    params.set("country", profile.country);
  }

  const data = await request<ListResponse<AdListItem>>(
    `/api/v1/ad?${params.toString()}`,
  );
  return data.items;
}

export function adClickUrl(adID: string) {
  const publicApiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  return `${publicApiUrl}/api/v1/ads/${encodeURIComponent(adID)}/click`;
}

export async function activateAdvertiser() {
  const token = await getSessionToken();
  return request<User>("/api/v1/advertiser/activate", {
    method: "POST",
    token,
  });
}

export async function listAdvertiserAds() {
  const token = await getSessionToken();
  const data = await request<ListResponse<AdvertiserAd>>("/api/v1/advertiser/ads", {
    token,
  });
  return data.items;
}

export async function getAdvertiserAnalytics() {
  const token = await getSessionToken();
  return request<AnalyticsSummary>("/api/v1/advertiser/analytics", { token });
}

export async function uploadImage(image: File) {
  const token = await getSessionToken();
  const form = new FormData();
  form.set("image", image);
  return request<Media>("/api/v1/media/images", {
    method: "POST",
    token,
    body: form,
  });
}

export async function createAdvertiserAd(input: {
  title: string;
  description?: string;
  landingPageUrl?: string;
  bid?: number;
  dailyBudget?: number;
  startAt: string;
  endAt: string;
  imageMediaId?: number;
  conditions?: {
    ageStart?: number;
    ageEnd?: number;
    country?: string[];
    platform?: string[];
  };
}) {
  const token = await getSessionToken();
  return request<AdvertiserAd>("/api/v1/advertiser/ads", {
    method: "POST",
    token,
    body: JSON.stringify(input),
  });
}
