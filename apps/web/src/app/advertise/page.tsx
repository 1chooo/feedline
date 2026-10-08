import Link from "next/link";
import { getCreditPackages, getCurrentUser, getMarketingSummary } from "@/lib/api";
import { authHref } from "@/lib/navigation";

const number = new Intl.NumberFormat("en-US");
const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export default async function AdvertisePage() {
  const [userResult, packagesResult, summaryResult] = await Promise.allSettled([
    getCurrentUser(),
    getCreditPackages(),
    getMarketingSummary(),
  ]);
  const user = userResult.status === "fulfilled" ? userResult.value : null;
  const packages = packagesResult.status === "fulfilled" ? packagesResult.value : [];
  const summary = summaryResult.status === "fulfilled" ? summaryResult.value : null;
  const workspaceHref = user ? "/advertiser" : authHref("signup", "/advertiser");

  return (
    <main className="min-h-svh bg-[#f5f5f1] text-[#142019] dark:bg-[#102019] dark:text-[#eff7ef]">
      <div className="min-h-full bg-[radial-gradient(circle_at_85%_0%,#d7f2b7_0,transparent_30rem)] dark:bg-[radial-gradient(circle_at_85%_0%,#284d37_0,transparent_30rem)]">
        <header className="mx-auto flex w-full max-w-7xl items-center justify-between px-5 py-5 sm:px-8">
          <Link href="/" className="text-lg font-semibold tracking-tight">Stream</Link>
          <nav className="hidden items-center gap-6 text-sm text-[#41534a] md:flex dark:text-[#b9c9bc]">
            <a href="#solutions">Solutions</a><a href="#pricing">Pricing</a><a href="#how-it-works">How it works</a><a href="#faq">FAQ</a>
          </nav>
          <Link href={workspaceHref} className="rounded-full bg-[#15251b] px-4 py-2 text-sm font-medium text-white dark:bg-[#e3f1df] dark:text-[#15251b]">{user ? "Advertiser workspace" : "Start advertising"}</Link>
        </header>

        <section className="mx-auto grid max-w-7xl gap-10 px-5 pb-20 pt-12 sm:px-8 lg:grid-cols-[1.05fr_.95fr] lg:items-center lg:pb-28 lg:pt-20">
          <div>
            <p className="inline-flex rounded-full border border-[#aac998] bg-white/55 px-3 py-1 text-xs font-medium uppercase tracking-[.16em] text-[#3f6b43] dark:border-[#527a5d] dark:bg-[#173323]/60 dark:text-[#b8e1ac]">Stream for business</p>
            <h1 className="mt-6 max-w-3xl text-5xl font-semibold leading-[.96] tracking-[-.055em] sm:text-6xl lg:text-7xl">Turn attention into a relationship.</h1>
            <p className="mt-6 max-w-xl text-lg leading-8 text-[#526259] dark:text-[#bfcebf]">Place clear, relevant campaigns inside a community feed—then see what was delivered, clicked, and funded from one advertiser workspace.</p>
            <div className="mt-8 flex flex-wrap gap-3"><Link href={workspaceHref} className="rounded-full bg-[#15251b] px-5 py-3 text-sm font-medium text-white dark:bg-[#e3f1df] dark:text-[#15251b]">{user ? "Open workspace" : "Create advertiser account"}</Link><a href="#pricing" className="rounded-full border border-[#9eb3a3] px-5 py-3 text-sm font-medium">Explore credits</a></div>
          </div>
          <DashboardPreview summary={summary} />
        </section>
      </div>

      <section className="bg-[#15251b] px-5 py-16 text-[#edf5e9] sm:px-8 lg:py-24">
        <div className="mx-auto max-w-7xl"><p className="text-sm font-medium text-[#b8e1ac]">Why Stream</p><div className="mt-5 grid gap-8 md:grid-cols-3"><Value title="Native by design" text="Image campaigns live naturally in the feed, with a clearly labeled destination and no surprise redirects." /><Value title="Audience controls" text="Set country, platform, and age ranges when they matter; leave targeting broad when they do not." /><Value title="Visible economics" text="Purchase credits, apply eligible promotions, and see campaign credit commitments in an immutable ledger." /></div></div>
      </section>

      <section id="solutions" className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:py-24"><SectionIntro eyebrow="Advertising solutions" title="A practical toolkit for launching a focused campaign." body="Stream supports the creative, audience, scheduling, and measurement workflow behind every campaign shown below." /><div className="mt-10 grid gap-4 lg:grid-cols-3"><Solution number="01" title="Feed placements" text="Use a visual creative, message, and landing-page link in the social feed." /><Solution number="02" title="Audience conditions" text="Reach by country, platform, and optional age range—with explicit exclusions supported by the API." /><Solution number="03" title="Delivery reporting" text="Review impressions, clicks, and click-through rate by day in the advertiser workspace." /></div></section>

      <section id="how-it-works" className="bg-white px-5 py-16 dark:bg-[#17281d] sm:px-8 lg:py-24"><div className="mx-auto max-w-7xl"><SectionIntro eyebrow="How it works" title="From business account to live delivery in four clear steps." /><ol className="mt-10 grid gap-5 md:grid-cols-4">{[["1", "Activate", "Open an advertiser workspace and name your company."], ["2", "Fund", "Choose a credit package or redeem an eligible credit code."], ["3", "Build", "Upload creative, set a schedule and audience, then allocate campaign credits."], ["4", "Learn", "Track delivery and engagement while campaign and credit records stay linked."]].map(([step, title, text]) => <li key={step} className="rounded-2xl border border-border bg-background p-5 dark:bg-[#102019]"><span className="text-sm font-medium text-[#4f8a54]">{step}</span><h3 className="mt-6 text-xl font-semibold">{title}</h3><p className="mt-2 text-sm leading-6 text-muted">{text}</p></li>)}</ol></div></section>

      <section id="pricing" className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:py-24"><SectionIntro eyebrow="Pricing & credits" title="Start with a credit package that fits your campaign." body="Credits are prepaid units: packages add to your company balance, and launching a funded campaign records its credit commitment." />{packagesResult.status === "rejected" ? <p className="mt-8 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-600 dark:border-red-950 dark:bg-red-950/30">Pricing is temporarily unavailable. Please try again shortly.</p> : null}{packages.length === 0 && packagesResult.status === "fulfilled" ? <p className="mt-8 rounded-xl border border-dashed border-border p-6 text-sm text-muted">No credit packages are currently available.</p> : null}<div className="mt-10 grid gap-4 md:grid-cols-3">{packages.map((creditPackage) => <article key={creditPackage.id} className="rounded-2xl border border-border bg-surface p-6"><p className="text-sm font-medium text-muted">{creditPackage.name}</p><p className="mt-3 text-3xl font-semibold tracking-tight">{money.format(creditPackage.priceCents / 100)}</p><p className="mt-5 text-lg font-medium">{number.format(creditPackage.credits)} credits</p>{creditPackage.bonusCredits ? <p className="mt-1 text-sm text-[#3f7c49]">+ {number.format(creditPackage.bonusCredits)} bonus credits</p> : <p className="mt-1 text-sm text-muted">No bonus credits</p>}<Link href={user ? `/advertiser?package=${creditPackage.id}` : authHref("signup", `/advertiser?package=${creditPackage.id}`)} className="mt-7 inline-flex rounded-full border border-border px-4 py-2 text-sm font-medium">Choose {creditPackage.name}</Link></article>)}</div><p className="mt-5 text-sm text-muted">Coupon and event promotions are applied according to their active dates, redemption limits, and reward rules. You can review each result in your company ledger.</p></section>

      <section className="bg-[#dff2ce] px-5 py-16 dark:bg-[#1c3d28] sm:px-8 lg:py-24"><div className="mx-auto max-w-7xl"><SectionIntro eyebrow="Audience & reach" title="Know the activity behind your campaign." body="These aggregate figures are generated from current Stream records, not static marketing copy." /><div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">{summary ? [["Monthly active accounts", number.format(summary.monthlyActiveUsers)], ["Registered advertisers", number.format(summary.activeAdvertisers)], ["Live campaigns", number.format(summary.activeCampaigns)], ["30-day impressions", number.format(summary.impressions30d)]].map(([label, value]) => <div key={label} className="rounded-2xl bg-white/70 p-5 dark:bg-[#102019]/70"><p className="text-sm text-muted">{label}</p><p className="mt-2 text-3xl font-semibold tracking-tight">{value}</p></div>) : <p className="rounded-2xl border border-dashed border-[#7aa77f] p-5 text-sm text-[#365b3c] dark:text-[#c8e7cb]">Reach metrics are temporarily unavailable. The advertiser dashboard remains the source of campaign-level reporting.</p>}</div></div></section>

      <section className="mx-auto max-w-7xl px-5 py-16 sm:px-8 lg:py-24">
        <SectionIntro eyebrow="Advertiser workspace" title="A clear view of the next campaign." body="Plan campaigns, monitor delivery, and follow every credit commitment in one place." />
        <div className="mt-8 grid gap-4 md:grid-cols-3">
          <Solution number="Plan" title="Campaign controls" text="Set creative, audience conditions, a UTC schedule, and a daily impression cap." />
          <Solution number="Measure" title="Daily reporting" text="Select your reporting dates and review impressions and clicks with accessible daily data." />
          <Solution number="Fund" title="Company ledger" text="Track available credits, purchases, promotional awards, and campaign commitments." />
        </div>
        <Link href={workspaceHref} className="mt-6 inline-flex rounded-full bg-foreground px-5 py-3 text-sm font-medium text-background">Explore your workspace</Link>
      </section>

      <section id="faq" className="bg-white px-5 py-16 dark:bg-[#17281d] sm:px-8 lg:py-24"><div className="mx-auto max-w-3xl"><SectionIntro eyebrow="FAQ" title="Common questions, answered directly." />{[["How do credits work?", "Packages add credits to the company balance. When a newly created campaign is funded, its allocated credits are recorded as campaign spend in the ledger before it is activated."], ["Can I use a promotional code?", "Yes. Bonus-credit codes can be redeemed to the balance, while eligible discount codes can be entered during a package purchase. Availability is controlled by active dates and redemption limits."], ["What performance data is available?", "Advertisers can view impressions, clicks, daily results, and click-through rate for campaigns they own."], ["Can I control the audience?", "Campaigns support country, platform, and age-range conditions. Every campaign remains visibly labeled when it appears in the feed."], ["Does Stream process production card payments yet?", "The development environment includes a clearly labeled manual checkout for workflow testing. Production checkout requires a configured payment provider before it can complete purchases."]].map(([question, answer]) => <details key={question} className="border-b border-border py-5"><summary className="cursor-pointer list-none pr-8 text-lg font-medium">{question}<span aria-hidden="true" className="float-right text-muted">+</span></summary><p className="mt-3 max-w-2xl text-sm leading-6 text-muted">{answer}</p></details>)}</div></section>

      <section className="bg-[#15251b] px-5 py-16 text-[#edf5e9] sm:px-8 lg:py-24"><div className="mx-auto flex max-w-7xl flex-col items-start justify-between gap-7 md:flex-row md:items-end"><div><p className="text-sm font-medium text-[#b8e1ac]">Ready when your message is.</p><h2 className="mt-3 max-w-2xl text-4xl font-semibold tracking-[-.04em] sm:text-5xl">Build your next campaign with a clear view of every credit and result.</h2></div><Link href={workspaceHref} className="shrink-0 rounded-full bg-[#e3f1df] px-5 py-3 text-sm font-medium text-[#15251b]">{user ? "Go to workspace" : "Start advertising"}</Link></div></section>
    </main>
  );
}

function DashboardPreview({ summary }: { summary: Awaited<ReturnType<typeof getMarketingSummary>> | null }) {
  return (
    <aside aria-label="Current platform activity" className="rounded-[2rem] border border-[#a5c89d] bg-surface p-5 shadow-xl dark:border-[#477654]">
      <p className="text-sm font-medium text-muted">Stream at a glance</p>
      <h2 className="mt-6 text-2xl font-semibold tracking-tight">Activity behind the feed</h2>
      {summary ? <dl className="mt-6 space-y-4">{[
        ["Impressions · last 30 days", summary.impressions30d],
        ["Monthly active accounts", summary.monthlyActiveUsers],
        ["Posts · last 30 days", summary.posts30d],
      ].map(([label, value]) => <div key={String(label)} className="flex flex-wrap items-baseline justify-between gap-2 border-b border-border pb-3"><dt className="text-sm text-muted">{label}</dt><dd className="text-2xl font-semibold">{number.format(Number(value))}</dd></div>)}</dl> : <p className="mt-6 text-sm text-muted">Activity metrics are temporarily unavailable.</p>}
      <p className="mt-5 text-xs leading-5 text-muted">Current aggregate activity. Your campaign results appear in your private workspace.</p>
    </aside>
  );
}

function SectionIntro({ eyebrow, title, body }: { eyebrow: string; title: string; body?: string }) {
  return <div><p className="text-sm font-medium text-[#4b854e] dark:text-[#a9d9a3]">{eyebrow}</p><h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-[-.04em] sm:text-4xl">{title}</h2>{body ? <p className="mt-4 max-w-2xl text-base leading-7 text-muted">{body}</p> : null}</div>;
}

function Value({ title, text }: { title: string; text: string }) {
  return <div><h2 className="text-2xl font-semibold tracking-tight">{title}</h2><p className="mt-3 max-w-sm text-sm leading-6 text-[#b6cdb7]">{text}</p></div>;
}

function Solution({ number: step, title, text }: { number: string; title: string; text: string }) {
  return <article className="rounded-2xl border border-border bg-surface p-6"><span className="text-sm font-medium text-[#4f8a54]">{step}</span><h3 className="mt-8 text-xl font-semibold">{title}</h3><p className="mt-3 text-sm leading-6 text-muted">{text}</p></article>;
}
