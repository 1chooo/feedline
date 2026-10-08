"use client";

import { useActionState } from "react";
import { fundCampaignAction, type AdvertiserState } from "@/app/actions/advertiser";

const initialState: AdvertiserState = {};

export function CampaignFundingForm({ campaignID }: { campaignID: number }) {
  const action = fundCampaignAction.bind(null, campaignID);
  const [state, formAction, pending] = useActionState(action, initialState);

  return (
    <form action={formAction} className="mt-3 flex flex-wrap items-end gap-2 rounded-xl bg-background p-3">
      <label className="min-w-32 flex-1 text-xs font-medium text-muted">
        Credits to launch
        <input
          name="credits"
          type="number"
          min="1"
          required
          className="mt-1 w-full rounded-lg border border-border bg-surface px-2.5 py-1.5 text-sm text-foreground outline-none"
        />
      </label>
      <button
        type="submit"
        disabled={pending}
        className="rounded-full bg-foreground px-3 py-2 text-xs font-medium text-background disabled:opacity-60"
      >
        {pending ? "Funding…" : "Fund & activate"}
      </button>
      {state.error ? <p className="w-full text-xs text-red-500">{state.error}</p> : null}
      {state.ok ? <p className="w-full text-xs text-green-600">{state.message}</p> : null}
    </form>
  );
}
