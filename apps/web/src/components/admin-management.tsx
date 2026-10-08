"use client";

import { useActionState } from "react";
import {
  adjustCreditsAction,
  changeCampaignStatusAction,
  changeUserRoleAction,
  createPromotionAction,
  togglePromotionAction,
  type AdminState,
} from "@/app/actions/admin";
import type { AdminCampaign, AdminCompany, AdminUser, Promotion } from "@/lib/api";

const initialState: AdminState = {};

export function AdminManagement({
  users,
  companies,
  campaigns,
  promotions,
}: {
  users: AdminUser[];
  companies: AdminCompany[];
  campaigns: AdminCampaign[];
  promotions: Promotion[];
}) {
  return (
    <div className="mt-8 space-y-6">
      <section className="rounded-2xl border border-border bg-surface p-5">
        <div>
          <p className="text-sm font-medium text-muted">Promotions</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight">Create and control offers</h2>
        </div>
        <PromotionForm />
        <div tabIndex={0} aria-label="Promotion management table" className="mt-5 overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead className="border-b border-border text-xs uppercase tracking-wide text-muted">
              <tr><th scope="col" className="pb-2 font-medium">Code</th><th scope="col" className="pb-2 font-medium">Offer</th><th scope="col" className="pb-2 font-medium">Redeemed</th><th scope="col" className="pb-2 font-medium">State</th></tr>
            </thead>
            <tbody className="divide-y divide-border">
              {promotions.map((promotion) => <PromotionRow key={promotion.id} promotion={promotion} />)}
              {promotions.length === 0 ? <tr><td colSpan={4} className="py-4 text-muted">No promotions configured.</td></tr> : null}
            </tbody>
          </table>
        </div>
      </section>

      <section className="rounded-2xl border border-border bg-surface p-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div><p className="text-sm font-medium text-muted">Advertisers</p><h2 className="mt-1 text-xl font-semibold tracking-tight">Companies and credit controls</h2></div>
          <p className="text-xs text-muted">Every adjustment is retained in the company ledger.</p>
        </div>
        <div tabIndex={0} aria-label="Company management table" className="mt-4 overflow-x-auto">
          <table className="w-full min-w-[760px] text-left text-sm">
            <thead className="border-b border-border text-xs uppercase tracking-wide text-muted">
              <tr><th scope="col" className="pb-2 font-medium">Company</th><th scope="col" className="pb-2 font-medium">Owner</th><th scope="col" className="pb-2 font-medium">Balance</th><th scope="col" className="pb-2 font-medium">Manual adjustment</th></tr>
            </thead>
            <tbody className="divide-y divide-border">
              {companies.map((company) => <CompanyRow key={company.id} company={company} />)}
              {companies.length === 0 ? <tr><td colSpan={4} className="py-4 text-muted">No advertiser companies yet.</td></tr> : null}
            </tbody>
          </table>
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-2xl border border-border bg-surface p-5">
          <p className="text-sm font-medium text-muted">User access</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight">Roles</h2>
          <div tabIndex={0} className="mt-4 max-h-[34rem] overflow-auto">
            <table className="w-full min-w-[520px] text-left text-sm">
              <thead className="sticky top-0 border-b border-border bg-surface text-xs uppercase tracking-wide text-muted"><tr><th scope="col" className="pb-2 font-medium">User</th><th scope="col" className="pb-2 font-medium">Role</th></tr></thead>
              <tbody className="divide-y divide-border">{users.map((user) => <UserRow key={user.id} user={user} />)}</tbody>
            </table>
          </div>
        </section>
        <section className="rounded-2xl border border-border bg-surface p-5">
          <p className="text-sm font-medium text-muted">Campaign operations</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight">Campaign status</h2>
          <div tabIndex={0} className="mt-4 max-h-[34rem] overflow-auto">
            <table className="w-full min-w-[580px] text-left text-sm">
              <thead className="sticky top-0 border-b border-border bg-surface text-xs uppercase tracking-wide text-muted"><tr><th scope="col" className="pb-2 font-medium">Campaign</th><th scope="col" className="pb-2 font-medium">Status</th></tr></thead>
              <tbody className="divide-y divide-border">{campaigns.map((campaign) => <CampaignRow key={campaign.id} campaign={campaign} />)}</tbody>
            </table>
          </div>
        </section>
      </div>
    </div>
  );
}

function PromotionForm() {
  const [state, action, pending] = useActionState(createPromotionAction, initialState);
  return (
    <form action={action} className="mt-4 grid gap-3 rounded-xl bg-background p-3 md:grid-cols-6">
      <input name="code" aria-label="Promotion code" required placeholder="WELCOME500" className={fieldClass} />
      <input name="name" aria-label="Promotion name" required placeholder="Offer name" className={fieldClass} />
      <select name="kind" aria-label="Promotion type" defaultValue="coupon" className={fieldClass}><option value="coupon">Coupon</option><option value="event">Event</option><option value="purchase">Purchase</option></select>
      <select name="rewardType" aria-label="Promotion reward type" defaultValue="bonus_credits" className={fieldClass}><option value="bonus_credits">Bonus credits</option><option value="percent_discount">Percent discount</option></select>
      <input name="rewardValue" aria-label="Promotion reward value" type="number" min="1" required placeholder="Value" className={fieldClass} />
      <div className="flex gap-2"><input name="maxRedemptions" aria-label="Maximum redemptions" type="number" min="1" placeholder="Max uses" className={`${fieldClass} min-w-0`} /><button type="submit" disabled={pending} className="rounded-full bg-foreground px-3 text-xs font-medium text-background disabled:opacity-60">{pending ? "…" : "Create"}</button></div>
      <StateMessage state={state} className="md:col-span-6" />
    </form>
  );
}

function PromotionRow({ promotion }: { promotion: Promotion }) {
  const action = togglePromotionAction.bind(null, promotion.id, !promotion.active);
  const [state, formAction, pending] = useActionState(action, initialState);
  return <tr>
    <td className="py-3 font-medium">{promotion.code}</td>
    <td className="py-3 capitalize">{promotion.rewardValue.toLocaleString()} {promotion.rewardType.replaceAll("_", " ")}</td>
    <td className="py-3">{promotion.totalRedemptions}{promotion.maxRedemptions ? ` / ${promotion.maxRedemptions}` : ""}</td>
    <td className="py-3"><form action={formAction} className="flex items-center gap-2"><button type="submit" disabled={pending} className="rounded-full border border-border px-2.5 py-1 text-xs font-medium disabled:opacity-60">{promotion.active ? "Pause" : "Activate"}</button><StateMessage state={state} /></form></td>
  </tr>;
}

function CompanyRow({ company }: { company: AdminCompany }) {
  const action = adjustCreditsAction.bind(null, company.id);
  const [state, formAction, pending] = useActionState(action, initialState);
  return <tr>
    <td className="py-3"><p className="font-medium">{company.name}</p><p className="text-xs text-muted">#{company.id}</p></td>
    <td className="py-3"><p>@{company.ownerUsername}</p><p className="text-xs text-muted">{company.ownerEmail}</p></td>
    <td className="py-3 font-medium">{company.creditBalance.toLocaleString()}</td>
    <td className="py-3"><form action={formAction} className="flex flex-wrap gap-2"><input name="deltaCredits" aria-label="Credit adjustment amount" type="number" step="1" required placeholder="+250 / -250" className={smallFieldClass} /><input name="note" aria-label="Credit adjustment reason" required maxLength={280} placeholder="Reason" className={smallFieldClass} /><button type="submit" disabled={pending} className="rounded-full border border-border px-2.5 py-1 text-xs font-medium disabled:opacity-60">{pending ? "…" : "Record"}</button><StateMessage state={state} /></form></td>
  </tr>;
}

function UserRow({ user }: { user: AdminUser }) {
  const action = changeUserRoleAction.bind(null, user.id);
  const [state, formAction, pending] = useActionState(action, initialState);
  return <tr><td className="py-3"><p className="font-medium">{user.displayName}</p><p className="text-xs text-muted">@{user.username} · {user.email}</p></td><td className="py-3"><form action={formAction} className="flex items-center gap-2"><select name="role" aria-label="User role" defaultValue={user.role} className={smallFieldClass}><option value="member">Member</option><option value="advertiser">Advertiser</option><option value="admin">Admin</option></select><button type="submit" disabled={pending} className="rounded-full border border-border px-2.5 py-1 text-xs font-medium disabled:opacity-60">Save</button><StateMessage state={state} /></form></td></tr>;
}

function CampaignRow({ campaign }: { campaign: AdminCampaign }) {
  const action = changeCampaignStatusAction.bind(null, campaign.id);
  const [state, formAction, pending] = useActionState(action, initialState);
  return <tr><td className="py-3"><p className="font-medium">{campaign.title}</p><p className="text-xs text-muted">{campaign.companyName || `@${campaign.advertiserUsername || "unassigned"}`} · {campaign.creditSpent.toLocaleString()} credits</p></td><td className="py-3"><form action={formAction} className="flex items-center gap-2"><select name="status" aria-label="Campaign status" defaultValue={campaign.status} className={smallFieldClass}><option value="active">Active</option><option value="paused">Paused</option><option value="archived">Archived</option><option value="canceled">Canceled</option></select><button type="submit" disabled={pending} className="rounded-full border border-border px-2.5 py-1 text-xs font-medium disabled:opacity-60">Save</button><StateMessage state={state} /></form></td></tr>;
}

function StateMessage({ state, className = "" }: { state: AdminState; className?: string }) {
  if (state.error) return <p role="alert" className={`text-xs text-red-500 ${className}`}>{state.error}</p>;
  if (state.ok) return <p role="status" className={`text-xs text-green-600 ${className}`}>{state.message}</p>;
  return null;
}

const fieldClass = "min-w-0 rounded-lg border border-border bg-surface px-2.5 py-2 text-sm outline-none";
const smallFieldClass = "min-w-0 rounded-lg border border-border bg-background px-2 py-1.5 text-xs outline-none";
