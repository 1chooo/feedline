"use server";

import { revalidatePath } from "next/cache";
import { ApiError, createPost } from "@/lib/api";

export type ComposeState = {
  error?: string;
  ok?: boolean;
};

export async function createPostAction(
  _prev: ComposeState,
  formData: FormData,
): Promise<ComposeState> {
  const title = String(formData.get("title") ?? "").trim();
  const description = String(formData.get("description") ?? "").trim();
  const imageUrl = String(formData.get("imageUrl") ?? "").trim();
  const landingPageUrl = String(formData.get("landingPageUrl") ?? "").trim();

  try {
    await createPost({
      title,
      description: description || undefined,
      imageUrl: imageUrl || undefined,
      landingPageUrl: landingPageUrl || undefined,
    });
  } catch (error) {
    if (error instanceof ApiError) {
      return { error: error.message };
    }
    return { error: "Could not publish. Is the API running?" };
  }

  revalidatePath("/");
  return { ok: true };
}
