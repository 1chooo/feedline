"use server";

import { revalidatePath } from "next/cache";
import {
  adjustAdminCredits,
  ApiError,
  createAdminPromotion,
  getCurrentUser,
  setAdminCampaignStatus,
  setAdminPromotionActive,
  setAdminUserRole,
  type AdminCampaign,
  type AdminUser,
  type Promotion,
} from "@/lib/api";

export type AdminState = { error?: string; ok?: boolean; message?: string };

async function runAdminMutation(task: () => Promise<unknown>): Promise<AdminState | null> {
  const user = await getCurrentUser();
  if (user?.role !== "admin") {
    return { error: "Administrator access is required." };
  }
  try {
    await task();
  } catch (error) {
    return { error: error instanceof ApiError ? error.message : "Could not update the platform." };
  }
  revalidatePath("/admin");
  return null;
}

export async function changeUserRoleAction(
  userID: number,
  _previous: AdminState,
  formData: FormData,
): Promise<AdminState> {
  const role = String(formData.get("role") ?? "") as AdminUser["role"];
  const failure = await runAdminMutation(() => setAdminUserRole(userID, role));
  return failure ?? { ok: true, message: "Role updated." };
}

export async function changeCampaignStatusAction(
  campaignID: number,
  _previous: AdminState,
  formData: FormData,
): Promise<AdminState> {
  const status = String(formData.get("status") ?? "") as AdminCampaign["status"];
  const failure = await runAdminMutation(() => setAdminCampaignStatus(campaignID, status));
  return failure ?? { ok: true, message: "Campaign status updated." };
}

export async function adjustCreditsAction(
  companyID: number,
  _previous: AdminState,
  formData: FormData,
): Promise<AdminState> {
  const deltaCredits = Number(formData.get("deltaCredits"));
  const note = String(formData.get("note") ?? "").trim();
  if (!Number.isInteger(deltaCredits) || deltaCredits === 0 || !note) {
    return { error: "Enter a non-zero whole credit amount and an adjustment note." };
  }
  const failure = await runAdminMutation(() => adjustAdminCredits(companyID, deltaCredits, note));
  return failure ?? { ok: true, message: "Credit adjustment recorded." };
}

export async function createPromotionAction(
  _previous: AdminState,
  formData: FormData,
): Promise<AdminState> {
  const code = String(formData.get("code") ?? "").trim();
  const name = String(formData.get("name") ?? "").trim();
  const rewardValue = Number(formData.get("rewardValue"));
  const max = String(formData.get("maxRedemptions") ?? "").trim();
  if (!code || !name || !Number.isInteger(rewardValue) || rewardValue < 1) {
    return { error: "Code, name, and a positive whole reward value are required." };
  }
  const failure = await runAdminMutation(() =>
    createAdminPromotion({
      code,
      name,
      kind: String(formData.get("kind") ?? "coupon") as Promotion["kind"],
      rewardType: String(formData.get("rewardType") ?? "bonus_credits") as Promotion["rewardType"],
      rewardValue,
      ...(max ? { maxRedemptions: Number(max) } : {}),
    }),
  );
  return failure ?? { ok: true, message: "Promotion created." };
}

export async function togglePromotionAction(
  promotionID: number,
  active: boolean,
  _previous: AdminState,
  _formData: FormData,
): Promise<AdminState> {
	void _previous;
	void _formData;
	const failure = await runAdminMutation(() => setAdminPromotionActive(promotionID, active));
  return failure ?? { ok: true, message: active ? "Promotion activated." : "Promotion paused." };
}
