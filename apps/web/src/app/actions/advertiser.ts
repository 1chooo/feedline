"use server";

import { revalidatePath } from "next/cache";
import {
  activateAdvertiser,
  ApiError,
  createAdvertiserAd,
  fundAdvertiserCampaign,
  purchaseCredits,
  redeemPromoCode,
  renameAdvertiserCompany,
  uploadImage,
} from "@/lib/api";

export type AdvertiserState = {
  error?: string;
  ok?: boolean;
  message?: string;
};

function messageOf(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  return "Something went wrong. Is the API running?";
}

export async function activateAdvertiserAction(): Promise<AdvertiserState> {
  try {
    await activateAdvertiser();
  } catch (error) {
    return { error: messageOf(error) };
  }

  revalidatePath("/");
  revalidatePath("/advertiser");
  return { ok: true };
}

export async function createCampaignAction(
  _previous: AdvertiserState,
  formData: FormData,
): Promise<AdvertiserState> {
  try {
    const title = requiredText(formData, "title", "Campaign name is required");
    const startAt = requiredDate(formData, "startAt", "Start date is required");
    const endAt = requiredDate(formData, "endAt", "End date is required");
		const credits = positiveInteger(formData, "credits", "Campaign credits are required");
    const image = formData.get("image");
    const media = image instanceof File && image.size > 0 ? await uploadImage(image) : undefined;

    const countries = String(formData.get("countries") ?? "")
      .split(",")
      .map((country) => country.trim().toUpperCase())
      .filter(Boolean);
    const platforms = formData
      .getAll("platform")
      .map((platform) => String(platform).trim())
      .filter(Boolean);
    const ageStart = optionalInteger(formData, "ageStart");
    const ageEnd = optionalInteger(formData, "ageEnd");
    const conditions =
      countries.length || platforms.length || ageStart !== undefined || ageEnd !== undefined
        ? {
            ...(countries.length ? { country: countries } : {}),
            ...(platforms.length ? { platform: platforms } : {}),
            ...(ageStart !== undefined ? { ageStart } : {}),
            ...(ageEnd !== undefined ? { ageEnd } : {}),
          }
        : undefined;

    const campaign = await createAdvertiserAd({
      title,
      description: optionalText(formData, "description"),
      landingPageUrl: optionalText(formData, "landingPageUrl"),
      bid: optionalNumber(formData, "bid"),
      dailyBudget: optionalInteger(formData, "dailyBudget"),
      startAt,
      endAt,
      imageMediaId: media?.id,
      conditions,
      status: "paused",
    });
    try {
      await fundAdvertiserCampaign(campaign.id, credits);
    } catch (error) {
      revalidatePath("/advertiser");
      return {
        error: `Campaign was saved as paused, but could not be funded: ${messageOf(error)}`,
      };
    }
  } catch (error) {
    return { error: error instanceof Error ? error.message : messageOf(error) };
  }

  revalidatePath("/advertiser");
  return { ok: true };
}

export async function fundCampaignAction(
  campaignID: number,
  _previous: AdvertiserState,
  formData: FormData,
): Promise<AdvertiserState> {
  try {
    await fundAdvertiserCampaign(
      campaignID,
      positiveInteger(formData, "credits", "Campaign credits are required"),
    );
  } catch (error) {
    return { error: messageOf(error) };
  }
  revalidatePath("/advertiser");
  return { ok: true, message: "Campaign funded and activated." };
}

export async function purchaseCreditsAction(
  _previous: AdvertiserState,
  formData: FormData,
): Promise<AdvertiserState> {
  try {
    await purchaseCredits(
      positiveInteger(formData, "packageId", "Choose a credit package"),
      optionalText(formData, "promoCode"),
    );
  } catch (error) {
    return { error: messageOf(error) };
  }
  revalidatePath("/advertiser");
  return { ok: true, message: "Credits added to your balance." };
}

export async function redeemPromoCodeAction(
  _previous: AdvertiserState,
  formData: FormData,
): Promise<AdvertiserState> {
  try {
    await redeemPromoCode(requiredText(formData, "promoCode", "Promo code is required"));
  } catch (error) {
    return { error: messageOf(error) };
  }
  revalidatePath("/advertiser");
  return { ok: true, message: "Promo code redeemed." };
}

export async function renameCompanyAction(
  _previous: AdvertiserState,
  formData: FormData,
): Promise<AdvertiserState> {
  try {
    await renameAdvertiserCompany(requiredText(formData, "companyName", "Company name is required"));
  } catch (error) {
    return { error: messageOf(error) };
  }
  revalidatePath("/advertiser");
  return { ok: true, message: "Company name updated." };
}

function requiredText(formData: FormData, field: string, message: string) {
  const value = optionalText(formData, field);
  if (!value) {
    throw new Error(message);
  }
  return value;
}

function optionalText(formData: FormData, field: string) {
  const value = String(formData.get(field) ?? "").trim();
  return value || undefined;
}

function requiredDate(formData: FormData, field: string, message: string) {
  const value = requiredText(formData, field, message);
  const date = new Date(`${value}:00.000Z`);
  if (Number.isNaN(date.valueOf())) {
    throw new Error(message);
  }
  return date.toISOString();
}

function optionalNumber(formData: FormData, field: string) {
  const value = optionalText(formData, field);
  if (!value) {
    return undefined;
  }
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    throw new Error(`${field} must be a number`);
  }
  return parsed;
}

function optionalInteger(formData: FormData, field: string) {
  const value = optionalNumber(formData, field);
  if (value === undefined) {
    return undefined;
  }
  if (!Number.isInteger(value)) {
    throw new Error(`${field} must be a whole number`);
  }
  return value;
}

function positiveInteger(formData: FormData, field: string, message: string) {
	const value = optionalInteger(formData, field);
	if (value === undefined || value < 1) {
		throw new Error(message);
	}
	return value;
}
