"use client";

import { useActionState } from "react";
import {
  activateAdvertiserAction,
  type AdvertiserState,
} from "@/app/actions/advertiser";

const initialState: AdvertiserState = {};

export function AdvertiserOnboarding() {
  const [state, action, pending] = useActionState(
    activateAdvertiserAction,
    initialState,
  );

  return (
    <section className="mx-auto mt-8 max-w-lg rounded-2xl border border-border bg-surface p-6">
      <p className="text-sm font-medium text-muted">Advertiser account</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight">Launch your first campaign</h1>
      <p className="mt-3 text-sm leading-6 text-muted">
        Add an advertiser workspace to your Stream account. You can continue posting
        in the community, and manage your company, credits, campaigns, and results here.
      </p>
      <form action={action} className="mt-6">
        {state.error ? <p role="alert" className="mb-3 text-sm text-red-500">{state.error}</p> : null}
        <button
          type="submit"
          disabled={pending}
          className="rounded-full bg-foreground px-5 py-2.5 text-sm font-medium text-background disabled:opacity-60"
        >
          {pending ? "Activating…" : "Activate advertiser account"}
        </button>
      </form>
    </section>
  );
}
