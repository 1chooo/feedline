"use client";

import { useActionState } from "react";
import {
  purchaseCreditsAction,
  redeemPromoCodeAction,
  renameCompanyAction,
  type AdvertiserState,
} from "@/app/actions/advertiser";
import type { BillingOverview, CreditPackage } from "@/lib/api";

const initialState: AdvertiserState = {};
const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export function AdvertiserBilling({ billing, selectedPackageId }: { billing: BillingOverview; selectedPackageId?: number }) {
  return (
    <section className="mt-8">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-muted">Credits & billing</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight">Fund your reach</h2>
        </div>
        <div className="rounded-2xl border border-border bg-surface px-4 py-3 text-right">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Available credits</p>
          <p className="mt-1 text-2xl font-semibold">{billing.company?.creditBalance.toLocaleString() ?? "0"}</p>
        </div>
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[1.2fr_.8fr]">
        <div className="rounded-2xl border border-border bg-surface p-5">
          <h3 className="font-semibold">Credit packages</h3>
          <p className="mt-1 text-sm text-muted">Buy credits first, then allocate them while you launch a campaign.</p>
          <p role="status" className="mt-3 rounded-xl border border-border bg-background px-3 py-2 text-sm text-muted">{billing.checkout?.message ?? "Checking credit checkout availability."}</p>
          {billing.packages.length === 0 ? (
            <p className="mt-4 rounded-xl border border-dashed border-border p-4 text-sm text-muted">
              Credit packages are not available right now.
            </p>
          ) : (
            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              {billing.packages.map((creditPackage) => (
                <PackageCard key={creditPackage.id} creditPackage={creditPackage} selected={creditPackage.id === selectedPackageId} checkoutEnabled={billing.checkout?.enabled ?? false} development={billing.checkout?.mode === "development"} />
              ))}
            </div>
          )}
        </div>
        <div className="space-y-4">
          <CompanyForm name={billing.company?.name ?? "Your company"} />
          <PromoForm />
        </div>
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <History title="Recent credit activity" empty="Credit activity will appear after your first purchase or campaign launch.">
          {billing.transactions.map((transaction) => (
            <li key={transaction.id} className="flex items-center justify-between gap-3 py-3 text-sm">
              <div>
                <p className="font-medium capitalize">{transaction.type.replaceAll("_", " ")}</p>
                <p className="mt-0.5 text-xs text-muted">{transaction.note || formatDate(transaction.createdAt)}</p>
              </div>
              <div className="text-right">
                <p className={transaction.deltaCredits > 0 ? "font-medium text-green-600" : "font-medium text-red-500"}>
                  {transaction.deltaCredits > 0 ? "+" : ""}{transaction.deltaCredits.toLocaleString()}
                </p>
                <p className="text-xs text-muted">{transaction.balanceAfter.toLocaleString()} balance</p>
              </div>
            </li>
          ))}
        </History>
        <History title="Purchase history" empty="Completed credit purchases will appear here.">
          {billing.purchases.map((purchase) => (
            <li key={purchase.id} className="flex items-center justify-between gap-3 py-3 text-sm">
              <div>
                <p className="font-medium">{purchase.credits.toLocaleString()} credits</p>
                <p className="mt-0.5 text-xs text-muted">{formatDate(purchase.createdAt)} · {purchase.status}</p>
              </div>
              <p className="font-medium">{money.format(purchase.amountCents / 100)}</p>
            </li>
          ))}
        </History>
      </div>
    </section>
  );
}

function PackageCard({ creditPackage, selected, checkoutEnabled, development }: { creditPackage: CreditPackage; selected: boolean; checkoutEnabled: boolean; development: boolean }) {
  const [state, action, pending] = useActionState(purchaseCreditsAction, initialState);
  return (
    <form action={action} aria-label={`${creditPackage.name} credit package`} className={`rounded-xl border bg-background p-3 ${selected ? "border-foreground ring-1 ring-foreground" : "border-border"}`}>
      <input type="hidden" name="packageId" value={creditPackage.id} />
      <p className="font-medium">{creditPackage.name}</p>
      {selected ? <p className="mt-1 text-xs font-medium">Your selected package</p> : null}
      <p className="mt-1 text-lg font-semibold">{money.format(creditPackage.priceCents / 100)}</p>
      <p className="mt-1 text-xs text-muted">
        {creditPackage.credits.toLocaleString()} credits
        {creditPackage.bonusCredits ? ` + ${creditPackage.bonusCredits.toLocaleString()} bonus` : ""}
      </p>
      <label className="mt-3 block text-xs text-muted">
        Promo code
        <input name="promoCode" className="mt-1 w-full rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-foreground" />
      </label>
      <button type="submit" disabled={pending || !checkoutEnabled} className="mt-3 w-full rounded-full bg-foreground px-3 py-2 text-xs font-medium text-background disabled:opacity-60">
        {pending ? "Processing…" : !checkoutEnabled ? "Checkout unavailable" : development ? "Add test credits" : "Buy credits"}
      </button>
      {state.error ? <p role="alert" className="mt-2 text-xs text-red-500">{state.error}</p> : null}
      {state.ok ? <p role="status" className="mt-2 text-xs text-green-600">{state.message}</p> : null}
    </form>
  );
}

function CompanyForm({ name }: { name: string }) {
  const [state, action, pending] = useActionState(renameCompanyAction, initialState);
  return (
    <form action={action} className="rounded-2xl border border-border bg-surface p-4">
      <p className="font-medium">Company</p>
      <label className="mt-3 block text-xs text-muted">
        Billing name
        <input name="companyName" defaultValue={name} required maxLength={120} className="mt-1 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground" />
      </label>
      <button type="submit" disabled={pending} className="mt-3 rounded-full border border-border px-3 py-1.5 text-xs font-medium disabled:opacity-60">
        {pending ? "Saving…" : "Save name"}
      </button>
      <FormMessage state={state} />
    </form>
  );
}

function PromoForm() {
  const [state, action, pending] = useActionState(redeemPromoCodeAction, initialState);
  return (
    <form action={action} className="rounded-2xl border border-border bg-surface p-4">
      <p className="font-medium">Have a bonus code?</p>
      <p className="mt-1 text-xs text-muted">Redeem credit promotions separately from a purchase.</p>
      <div className="mt-3 flex gap-2">
        <input name="promoCode" aria-label="Bonus credit promo code" required placeholder="Enter code" className="min-w-0 flex-1 rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground" />
        <button type="submit" disabled={pending} className="rounded-full border border-border px-3 py-2 text-xs font-medium disabled:opacity-60">
          {pending ? "Redeeming…" : "Redeem"}
        </button>
      </div>
      <FormMessage state={state} />
    </form>
  );
}

function History({ title, empty, children }: { title: string; empty: string; children: React.ReactNode }) {
  const items = Array.isArray(children) ? children : [children];
  return (
    <section className="rounded-2xl border border-border bg-surface p-5">
      <h3 className="font-semibold">{title}</h3>
      {items.length === 0 ? <p className="mt-3 text-sm text-muted">{empty}</p> : <ul className="mt-2 divide-y divide-border">{children}</ul>}
    </section>
  );
}

function FormMessage({ state }: { state: AdvertiserState }) {
  if (state.error) return <p role="alert" className="mt-2 text-xs text-red-500">{state.error}</p>;
  if (state.ok) return <p role="status" className="mt-2 text-xs text-green-600">{state.message}</p>;
  return null;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-US", { dateStyle: "medium" }).format(new Date(value));
}
