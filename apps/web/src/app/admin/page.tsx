import { redirect } from "next/navigation";
import { AdminManagement } from "@/components/admin-management";
import {
  getAdminAnalytics,
  getCurrentUser,
  listAdminCampaigns,
  listAdminCompanies,
  listAdminPromotions,
  listAdminUsers,
  type AdminAnalyticsSummary,
} from "@/lib/api";

const number = new Intl.NumberFormat("en-US");
const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 });

export default async function AdminPage() {
  const user = await getCurrentUser();
  if (!user) redirect("/login");
  if (user.role !== "admin") redirect("/");

  const results = await Promise.allSettled([
    getAdminAnalytics(),
    listAdminUsers(),
    listAdminCompanies(),
    listAdminCampaigns(),
    listAdminPromotions(),
  ]);
  const [analyticsResult, usersResult, companiesResult, campaignsResult, promotionsResult] = results;
  const analytics = analyticsResult.status === "fulfilled" ? analyticsResult.value : null;
  const loadError = results.some((result) => result.status === "rejected");

  return (
    <main className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 sm:py-10">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-muted">Platform administration</p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight">Business health</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted">Live records across members, content, advertisers, campaign delivery, credits, and promotions.</p>
        </div>
        {analytics ? <p className="text-sm text-muted">{analytics.from} – {analytics.to} UTC</p> : null}
      </div>

      {loadError ? <p className="mt-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600 dark:border-red-950 dark:bg-red-950/30">Some administrative data could not be loaded. Refresh when the API is available.</p> : null}
      {analytics ? <AnalyticsDashboard analytics={analytics} /> : null}
      {!analytics && !loadError ? <AdminLoading /> : null}

      <AdminManagement
        users={usersResult.status === "fulfilled" ? usersResult.value : []}
        companies={companiesResult.status === "fulfilled" ? companiesResult.value : []}
        campaigns={campaignsResult.status === "fulfilled" ? campaignsResult.value : []}
        promotions={promotionsResult.status === "fulfilled" ? promotionsResult.value : []}
      />
    </main>
  );
}

function AnalyticsDashboard({ analytics }: { analytics: AdminAnalyticsSummary }) {
  const cards = [
    ["Registered users", number.format(analytics.users.total), `${number.format(analytics.users.newRegistrations)} new in range`],
    ["Active users", number.format(analytics.users.activeInRange), `DAU ${number.format(analytics.users.dau)} · WAU ${number.format(analytics.users.wau)}`],
    ["Ad revenue", money.format(analytics.billing.revenueCents / 100), `${number.format(analytics.billing.purchases)} purchases`],
    ["Impressions", number.format(analytics.advertising.impressions), `${number.format(analytics.advertising.clicks)} clicks · ${(analytics.advertising.engagementRate * 100).toFixed(2)}% CTR`],
  ];
  return <>
    <section className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map(([label, value, detail]) => <div key={label} className="rounded-2xl border border-border bg-surface p-4"><p className="text-sm text-muted">{label}</p><p className="mt-2 text-2xl font-semibold tracking-tight">{value}</p><p className="mt-1 text-xs text-muted">{detail}</p></div>)}
    </section>
    <section className="mt-4 grid gap-4 xl:grid-cols-[1.25fr_.75fr]">
      <TrendChart daily={analytics.daily} />
      <div className="rounded-2xl border border-border bg-surface p-5"><p className="text-sm font-medium text-muted">Campaign lifecycle</p><h2 className="mt-1 text-xl font-semibold tracking-tight">Advertising operations</h2><dl className="mt-5 grid grid-cols-2 gap-x-4 gap-y-4 text-sm">{[
        ["Advertisers", analytics.advertising.advertisers], ["Companies", analytics.advertising.companies], ["Paying companies", analytics.advertising.payingCompanies], ["Active", analytics.advertising.activeCampaigns], ["Scheduled", analytics.advertising.scheduledCampaigns], ["Completed", analytics.advertising.completedCampaigns], ["Canceled", analytics.advertising.canceledCampaigns], ["Paused", analytics.advertising.pausedCampaigns],
      ].map(([label, value]) => <div key={String(label)}><dt className="text-muted">{label}</dt><dd className="mt-1 text-lg font-semibold">{number.format(Number(value))}</dd></div>)}</dl></div>
    </section>
    <section className="mt-4 grid gap-4 lg:grid-cols-3">
      <MetricPanel title="Retention" items={[["Day 1", `${analytics.retention.day1Percent.toFixed(1)}%`], ["Day 7", `${analytics.retention.day7Percent.toFixed(1)}%`], ["Day 30", `${analytics.retention.day30Percent.toFixed(1)}%`]]} />
      <MetricPanel title="Content & activity" items={[["Posts created", number.format(analytics.content.postsCreated)], ["Monthly active", number.format(analytics.users.mau)], ["New registrations", number.format(analytics.users.newRegistrations)]]} />
      <MetricPanel title="Credits & promotions" items={[["Credits purchased", number.format(analytics.billing.creditsPurchased)], ["Credits used", number.format(analytics.billing.creditsUsed)], ["Coupon redemptions", number.format(analytics.billing.couponRedemptions)], ["Promo credits", number.format(analytics.billing.promotionalCredits)]]} />
    </section>
  </>;
}

function TrendChart({ daily }: { daily: AdminAnalyticsSummary["daily"] }) {
  const max = Math.max(1, ...daily.map((point) => Math.max(point.activeUsers, point.impressions)));
  return <section className="rounded-2xl border border-border bg-surface p-5"><div className="flex items-end justify-between gap-3"><div><p className="text-sm font-medium text-muted">30-day activity</p><h2 className="mt-1 text-xl font-semibold tracking-tight">Users and delivery</h2></div><div className="flex gap-3 text-xs text-muted"><span><i className="mr-1 inline-block h-2 w-2 rounded-full bg-foreground" />Active users</span><span><i className="mr-1 inline-block h-2 w-2 rounded-full bg-muted" />Impressions</span></div></div><div className="mt-6 flex h-40 items-end gap-1" aria-label="Daily active users and advertising impressions chart">{daily.map((point) => <div key={point.date} className="group flex h-full min-w-0 flex-1 items-end gap-px" title={`${point.date}: ${point.activeUsers} active users, ${point.impressions} impressions`}><span className="w-1/2 rounded-t bg-foreground/80" style={{ height: `${Math.max(3, (point.activeUsers / max) * 100)}%` }} /><span className="w-1/2 rounded-t bg-muted" style={{ height: `${Math.max(3, (point.impressions / max) * 100)}%` }} /></div>)}</div><div className="mt-2 flex justify-between text-xs text-muted"><span>{daily.at(0)?.date}</span><span>{daily.at(-1)?.date}</span></div></section>;
}

function MetricPanel({ title, items }: { title: string; items: Array<[string, string]> }) {
  return <section className="rounded-2xl border border-border bg-surface p-5"><h2 className="font-semibold">{title}</h2><dl className="mt-4 space-y-3">{items.map(([label, value]) => <div key={label} className="flex items-baseline justify-between gap-4"><dt className="text-sm text-muted">{label}</dt><dd className="font-semibold">{value}</dd></div>)}</dl></section>;
}

function AdminLoading() {
  return <div className="mt-6 grid animate-pulse gap-3 sm:grid-cols-2 xl:grid-cols-4">{Array.from({ length: 4 }, (_, index) => <div key={index} className="h-28 rounded-2xl border border-border bg-surface" />)}</div>;
}
