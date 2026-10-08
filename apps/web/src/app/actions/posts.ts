"use server";

import { revalidatePath } from "next/cache";
import { ApiError, createPost, uploadImage } from "@/lib/api";

export type ComposeState = {
  error?: string;
  ok?: boolean;
  submissionId?: string;
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
    if (!title || title.length > 200 || description.length > 2000) return { error: "Enter a title up to 200 characters and a caption up to 2,000 characters." };
    const image = formData.get("image");
    if (image instanceof File && image.size > 5 * 1024 * 1024) return { error: "Choose an image of 5 MB or less." };
    const media = image instanceof File && image.size > 0 ? await uploadImage(image) : undefined;
    await createPost({
      title,
      description: description || undefined,
      imageUrl: imageUrl || undefined,
      imageMediaId: media?.id,
      landingPageUrl: landingPageUrl || undefined,
    });
  } catch (error) {
    if (error instanceof ApiError) {
      return { error: error.message };
    }
    return { error: "Could not publish your post. Your text is still here; please try again." };
  }

  revalidatePath("/");
  return { ok: true, submissionId: crypto.randomUUID() };
}
