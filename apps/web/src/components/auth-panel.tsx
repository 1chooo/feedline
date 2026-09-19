"use client";

import { useState } from "react";
import { SignInForm, SignUpForm } from "@/components/auth-forms";

export function AuthPanel() {
  const [mode, setMode] = useState<"choose" | "signin" | "signup">("choose");

  return (
    <aside className="sticky top-0 h-svh w-full max-w-80 px-6 py-10">
      <h2 className="text-xl font-semibold tracking-tight">
        {mode === "signup" ? "Create your Stream account" : "Log in or sign up for Stream"}
      </h2>
      <p className="mt-2 text-sm leading-6 text-muted">
        You can scroll the feed now. Sign in to post.
      </p>
      {mode === "choose" ? (
        <>
          <button
            type="button"
            onClick={() => setMode("signin")}
            className="mt-6 w-full rounded-full bg-foreground py-2.5 text-sm font-medium text-background"
          >
            Sign in
          </button>
          <button
            type="button"
            onClick={() => setMode("signup")}
            className="mt-3 w-full rounded-full border border-border py-2.5 text-sm font-medium"
          >
            Sign up
          </button>
        </>
      ) : (
        <div className="mt-6">
          {mode === "signin" ? (
            <SignInForm onSwitch={() => setMode("signup")} />
          ) : (
            <SignUpForm onSwitch={() => setMode("signin")} />
          )}
        </div>
      )}
    </aside>
  );
}
