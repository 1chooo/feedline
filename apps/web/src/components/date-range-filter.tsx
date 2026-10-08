import Link from "next/link";
import type { DateRange } from "@/lib/date-range";

export function DateRangeFilter({ range, path, packageId }: { range: DateRange; path: string; packageId?: number }) {
  return (
    <section aria-label="Reporting date range" className="mt-6 rounded-2xl border border-border bg-surface p-4">
      <form method="get" action={path} className="flex flex-wrap items-end gap-3">
        {packageId ? <input type="hidden" name="package" value={packageId} /> : null}
        <label className="block min-w-32 flex-1 text-sm">From<input name="from" type="date" required defaultValue={range.from} className="mt-1 w-full rounded-xl border border-border bg-background px-3 py-2" /></label>
        <label className="block min-w-32 flex-1 text-sm">To<input name="to" type="date" required defaultValue={range.to} className="mt-1 w-full rounded-xl border border-border bg-background px-3 py-2" /></label>
        <button type="submit" className="rounded-full bg-foreground px-4 py-2.5 text-sm font-medium text-background">Apply dates</button>
        <Link href={packageId ? `${path}?package=${packageId}` : path} className="rounded-full border border-border px-4 py-2.5 text-sm">Last 30 days</Link>
      </form>
      <p className="mt-2 text-xs text-muted">Inclusive dates in UTC · up to 90 days. Account and campaign totals show the current snapshot.</p>
      {range.error ? <p role="alert" className="mt-3 text-sm text-red-600 dark:text-red-400">{range.error}</p> : null}
    </section>
  );
}
