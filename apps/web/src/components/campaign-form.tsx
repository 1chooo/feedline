"use client";

import { useActionState, useEffect, useRef } from "react";
import {
  createCampaignAction,
  type AdvertiserState,
} from "@/app/actions/advertiser";

const initialState: AdvertiserState = {};
const fieldClass =
  "mt-1 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm outline-none";

export function CampaignForm() {
  const [state, action, pending] = useActionState(createCampaignAction, initialState);
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (state.ok) {
      formRef.current?.reset();
    }
  }, [state.ok]);

  return (
    <form
      ref={formRef}
      action={action}
      encType="multipart/form-data"
      className="rounded-2xl border border-border bg-surface p-5"
    >
      <div className="flex items-baseline justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">New campaign</h2>
          <p className="mt-1 text-sm text-muted">Creative files are stored in your configured media provider.</p>
        </div>
        <span className="text-xs text-muted">UTC schedule</span>
      </div>

      <div className="mt-5 grid gap-4 sm:grid-cols-2">
        <label className="block text-sm sm:col-span-2">
          Campaign name
          <input name="title" required maxLength={200} className={fieldClass} />
        </label>
        <label className="block text-sm sm:col-span-2">
          Description
          <textarea name="description" rows={3} maxLength={2000} className={`${fieldClass} resize-none`} />
        </label>
        <label className="block text-sm">
          Start
          <input name="startAt" type="datetime-local" required className={fieldClass} />
        </label>
        <label className="block text-sm">
          End
          <input name="endAt" type="datetime-local" required className={fieldClass} />
        </label>
        <label className="block text-sm sm:col-span-2">
          Landing page
          <input name="landingPageUrl" type="url" placeholder="https://example.com/launch" className={fieldClass} />
        </label>
        <label className="block text-sm">
          CPM bid
          <input name="bid" type="number" min="0" step="0.01" placeholder="2.50" className={fieldClass} />
        </label>
        <label className="block text-sm">
          Daily impression cap
          <input name="dailyBudget" type="number" min="0" step="1" placeholder="10000" className={fieldClass} />
        </label>
        <label className="block text-sm sm:col-span-2">
          Image creative
          <input
            name="image"
            type="file"
            accept="image/jpeg,image/png,image/webp,image/gif"
            className="mt-1 block w-full text-sm text-muted file:mr-3 file:rounded-full file:border-0 file:bg-background file:px-3 file:py-1.5 file:text-sm file:font-medium"
          />
          <span className="mt-1 block text-xs text-muted">JPEG, PNG, WebP, or GIF up to 5 MB.</span>
        </label>
      </div>

      <fieldset className="mt-5 border-t border-border pt-5">
        <legend className="text-sm font-medium">Audience (optional)</legend>
        <div className="mt-3 grid gap-4 sm:grid-cols-2">
          <label className="block text-sm">
            Countries
            <input name="countries" placeholder="US, TW, JP" className={fieldClass} />
          </label>
          <div className="text-sm">
            Platforms
            <div className="mt-2 flex flex-wrap gap-3 text-sm text-muted">
              {[
                ["web", "Web"],
                ["ios", "iOS"],
                ["android", "Android"],
              ].map(([value, label]) => (
                <label key={value} className="flex items-center gap-1.5">
                  <input name="platform" type="checkbox" value={value} />
                  {label}
                </label>
              ))}
            </div>
          </div>
          <label className="block text-sm">
            Minimum age
            <input name="ageStart" type="number" min="1" max="100" className={fieldClass} />
          </label>
          <label className="block text-sm">
            Maximum age
            <input name="ageEnd" type="number" min="1" max="100" className={fieldClass} />
          </label>
        </div>
      </fieldset>

      {state.error ? <p className="mt-4 text-sm text-red-500">{state.error}</p> : null}
      {state.ok ? <p className="mt-4 text-sm text-green-600">Campaign created.</p> : null}
      <div className="mt-5 flex justify-end">
        <button
          type="submit"
          disabled={pending}
          className="rounded-full bg-foreground px-5 py-2.5 text-sm font-medium text-background disabled:opacity-60"
        >
          {pending ? "Creating…" : "Create campaign"}
        </button>
      </div>
    </form>
  );
}
