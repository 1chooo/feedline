import { redirect } from "next/navigation";
import { AdvertiserOnboarding } from "@/components/advertiser-onboarding";
import { AdvertiserBilling } from "@/components/advertiser-billing";
import { CampaignForm } from "@/components/campaign-form";
import { CampaignFundingForm } from "@/components/campaign-funding-form";
import { DateRangeFilter } from "@/components/date-range-filter";
import { DailyChart } from "@/components/daily-chart";
import { reportingRange } from "@/lib/date-range";
import {
  getAdvertiserBilling,
  getAdvertiserAnalytics,
  getCurrentUser,
  listAdvertiserAds,
} from "@/lib/api";

import { authHref, selectedPackageID } from "@/lib/navigation";

const number = new Intl.NumberFormat("en-US");

export default async function AdvertiserPage({ searchParams }: PageProps<"/advertiser">) {
  const query = await searchParams;
  const packageID = selectedPackageID(query.package);
  const range = reportingRange(query.from, query.to);
  const destination = packageID ? `/advertiser?package=${packageID}` : "/advertiser";
  const user = await getCurrentUser();
  if (!user) {
    redirect(authHref("login", destination));
  }

  if (user.role !== "advertiser" && user.role !== "admin") {
    return <AdvertiserOnboarding />;
  }

  const results = await Promise.allSettled([
    listAdvertiserAds(),
    range.error ? Promise.resolve(null) : getAdvertiserAnalytics({ from: range.from, to: range.to }),
    getAdvertiserBilling(),
  ]);
  const [campaignResult, analyticsResult, billingResult] = results;
  const campaigns = campaignResult.status === "fulfilled" ? campaignResult.value : [];
  const analytics = analyticsResult.status === "fulfilled" ? analyticsResult.value : null;
  const billing = billingResult.status === "fulfilled" ? billingResult.value : null;
  const loadError = results.some((result) => result.status === "rejected") ? "Some workspace data could not be loaded. Refresh to try again." : null;

  if (billing && !billing.company) return <AdvertiserOnboarding />;

  return (
    <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:py-10">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-muted">Advertiser workspace</p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight">Campaigns</h1>
        </div>
        {analytics ? (
          <p className="text-sm text-muted">
            {analytics.from} – {analytics.to}
          </p>
        ) : null}
      </div>

      <DateRangeFilter key={`${range.from}-${range.to}`} range={range} path="/advertiser" packageId={packageID} />
      {loadError ? <p role="alert" className="mt-6 text-sm text-red-600 dark:text-red-400">{loadError}</p> : null}
      {analytics ? <AnalyticsCards analytics={analytics} /> : null}
      {analytics ? <section className="mt-4 grid gap-4 lg:grid-cols-2"><DailyChart title="Campaign impressions" data={analytics.daily.map((point) => ({ date: point.date, value: point.impressions }))} /><DailyChart title="Campaign clicks" data={analytics.daily.map((point) => ({ date: point.date, value: point.clicks }))} /></section> : null}
      {billing ? <AdvertiserBilling billing={billing} selectedPackageId={packageID} /> : null}
      <div className="mt-8">
        <CampaignForm />
      </div>
      <section className="mt-10">
        <h2 className="text-lg font-semibold">Your campaigns</h2>
        {campaigns.length === 0 ? (
          <p className="mt-3 rounded-2xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted">
            No campaigns yet. Create one above to start reaching people on Stream.
          </p>
        ) : (
          <div className="mt-3 overflow-hidden rounded-2xl border border-border bg-surface">
            {campaigns.map((campaign) => (
              <article key={campaign.id} className="border-b border-border p-4 last:border-b-0">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h3 className="font-medium">{campaign.title}</h3>
                    <p className="mt-1 text-sm text-muted">
                      {formatDate(campaign.startAt)} – {formatDate(campaign.endAt)}
                    </p>
                  </div>
                  <span className="rounded-full border border-border px-2.5 py-1 text-xs font-medium capitalize">
                    {campaign.status}
                  </span>
                </div>
                <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted">
                  <span>
                    {campaign.bid !== undefined ? `Delivery priority ${campaign.bid.toFixed(2)}` : "Default delivery priority"}
                  </span>
                  <span>
                    {campaign.dailyBudget !== undefined
                      ? `${number.format(campaign.dailyBudget)} daily impressions`
                      : "No daily cap"}
                  </span>
				  <span>
					{campaign.creditBudget !== undefined
						? `${number.format(campaign.creditSpent)} of ${number.format(campaign.creditBudget)} credits committed`
						: "Not yet funded"}
				  </span>
                </div>
				{campaign.status === "paused" && campaign.creditBudget === undefined ? (
				  <CampaignFundingForm campaignID={campaign.id} />
				) : null}
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}

function AnalyticsCards({
  analytics,
}: {
  analytics: Awaited<ReturnType<typeof getAdvertiserAnalytics>>;
}) {
  const stats = [
    ["Impressions", number.format(analytics.impressions)],
    ["Clicks", number.format(analytics.clicks)],
    ["Click-through rate", `${(analytics.ctr * 100).toFixed(2)}%`],
  ];
  return (
    <section className="mt-6 grid gap-3 sm:grid-cols-3">
      {stats.map(([label, value]) => (
        <div key={label} className="rounded-2xl border border-border bg-surface p-4">
          <p className="text-sm text-muted">{label}</p>
          <p className="mt-2 text-2xl font-semibold tracking-tight">{value}</p>
        </div>
      ))}
    </section>
  );
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en-US", { dateStyle: "medium" }).format(new Date(value));
}
