"use server";

import { redirect } from "next/navigation";
import { authDestination } from "@/lib/navigation";
import {
  ApiError,
  clearSessionCookie,
  login as loginRequest,
  logout as logoutRequest,
  register as registerRequest,
  setSessionCookie,
} from "@/lib/api";

export type AuthState = {
  error?: string;
};

function messageOf(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  return "Something went wrong. Is the API running?";
}

export async function loginAction(
  _prev: AuthState,
  formData: FormData,
): Promise<AuthState> {
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  try {
    const result = await loginRequest({ email, password });
    await setSessionCookie(result.token);
  } catch (error) {
    return { error: messageOf(error) };
  }

  redirect(authDestination(formData.get("next")));
}

export async function registerAction(
  _prev: AuthState,
  formData: FormData,
): Promise<AuthState> {
  const username = String(formData.get("username") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  const displayName = String(formData.get("displayName") ?? "").trim();
  const bio = String(formData.get("bio") ?? "").trim();

  try {
    const result = await registerRequest({
      username,
      email,
      password,
      displayName,
      bio: bio || undefined,
    });
    await setSessionCookie(result.token);
  } catch (error) {
    return { error: messageOf(error) };
  }

  redirect(authDestination(formData.get("next")));
}

export async function logoutAction() {
  await logoutRequest();
  await clearSessionCookie();
  redirect("/");
}
