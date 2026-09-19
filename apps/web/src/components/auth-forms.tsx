"use client";

import Link from "next/link";
import { useActionState } from "react";
import {
  loginAction,
  registerAction,
  type AuthState,
} from "@/app/actions/auth";

const initialState: AuthState = {};

const fieldClass =
  "mt-1 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-none";

export function SignInForm({
  nextHref = "/signup",
  onSwitch,
}: {
  nextHref?: string;
  onSwitch?: () => void;
}) {
  const [state, action, pending] = useActionState(loginAction, initialState);

  return (
    <form action={action} className="space-y-3">
      <label className="block text-sm">
        Email
        <input
          name="email"
          type="email"
          required
          autoComplete="email"
          className={fieldClass}
        />
      </label>
      <label className="block text-sm">
        Password
        <input
          name="password"
          type="password"
          required
          minLength={8}
          autoComplete="current-password"
          className={fieldClass}
        />
      </label>
      {state.error ? <p className="text-sm text-red-500">{state.error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="w-full rounded-full bg-foreground py-2.5 text-sm font-medium text-background disabled:opacity-60"
      >
        {pending ? "Signing in…" : "Sign in"}
      </button>
      <AuthSwitch
        prompt="New to Stream?"
        label="Sign up"
        href={nextHref}
        onSwitch={onSwitch}
      />
    </form>
  );
}

export function SignUpForm({
  nextHref = "/login",
  onSwitch,
}: {
  nextHref?: string;
  onSwitch?: () => void;
}) {
  const [state, action, pending] = useActionState(registerAction, initialState);

  return (
    <form action={action} className="space-y-3">
      <label className="block text-sm">
        Display name
        <input name="displayName" required autoComplete="name" className={fieldClass} />
      </label>
      <label className="block text-sm">
        Username
        <input
          name="username"
          required
          minLength={3}
          autoComplete="username"
          className={fieldClass}
        />
      </label>
      <label className="block text-sm">
        Email
        <input
          name="email"
          type="email"
          required
          autoComplete="email"
          className={fieldClass}
        />
      </label>
      <label className="block text-sm">
        Password
        <input
          name="password"
          type="password"
          required
          minLength={8}
          autoComplete="new-password"
          className={fieldClass}
        />
      </label>
      <label className="block text-sm">
        Bio
        <textarea name="bio" rows={2} className={`${fieldClass} resize-none`} />
      </label>
      {state.error ? <p className="text-sm text-red-500">{state.error}</p> : null}
      <button
        type="submit"
        disabled={pending}
        className="w-full rounded-full bg-foreground py-2.5 text-sm font-medium text-background disabled:opacity-60"
      >
        {pending ? "Creating account…" : "Sign up"}
      </button>
      <AuthSwitch
        prompt="Already have an account?"
        label="Sign in"
        href={nextHref}
        onSwitch={onSwitch}
      />
    </form>
  );
}

function AuthSwitch({
  prompt,
  label,
  href,
  onSwitch,
}: {
  prompt: string;
  label: string;
  href: string;
  onSwitch?: () => void;
}) {
  return (
    <p className="text-sm text-muted">
      {prompt}{" "}
      {onSwitch ? (
        <button
          type="button"
          onClick={onSwitch}
          className="font-medium underline-offset-2 hover:underline"
        >
          {label}
        </button>
      ) : (
        <Link href={href} className="font-medium underline-offset-2 hover:underline">
          {label}
        </Link>
      )}
    </p>
  );
}
