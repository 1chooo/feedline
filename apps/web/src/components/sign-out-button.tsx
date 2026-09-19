"use client";

import { logoutAction } from "@/app/actions/auth";

export function SignOutButton({ className }: { className?: string }) {
  return (
    <form action={logoutAction}>
      <button type="submit" className={className}>
        Sign out
      </button>
    </form>
  );
}
