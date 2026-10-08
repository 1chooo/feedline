const counts = new Intl.NumberFormat("en-US");
const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export function DailyChart({ title, data, currency = false }: { title: string; data: Array<{ date: string; value: number }>; currency?: boolean }) {
  const maximum = Math.max(0, ...data.map((point) => point.value));
  const format = (value: number) => currency ? money.format(value / 100) : counts.format(value);
  return (
    <figure className="min-w-0 rounded-2xl border border-border bg-surface p-5">
      <figcaption className="flex flex-wrap items-baseline justify-between gap-2"><h2 className="font-semibold">{title}</h2><span className="text-xs text-muted">Daily · UTC</span></figcaption>
      {maximum > 0 ? <>
        <p className="mt-3 text-right text-xs text-muted">Peak {format(maximum)}</p>
        <div aria-hidden="true" className="mt-2 flex h-32 items-end gap-1 border-b border-border">
          {data.map((point) => <div key={point.date} className="flex h-full min-w-0 flex-1 items-end" title={`${point.date}: ${format(point.value)}`}><div className="w-full rounded-t bg-foreground/75" style={{ height: `${point.value / maximum * 100}%` }} /></div>)}
        </div>
        <div aria-hidden="true" className="mt-2 flex justify-between gap-2 text-xs text-muted"><span>{data[0]?.date}</span><span>{data.at(-1)?.date}</span></div>
      </> : <p className="my-8 rounded-xl border border-dashed border-border p-5 text-center text-sm text-muted">No activity recorded for this reporting period.</p>}
      <details className="mt-4 text-sm"><summary className="cursor-pointer font-medium">View daily data: {title}</summary><div className="mt-3 max-h-64 overflow-auto"><table className="w-full text-left text-sm"><caption className="sr-only">{title} by date</caption><thead className="sticky top-0 bg-surface"><tr><th scope="col" className="py-2 font-medium">Date (UTC)</th><th scope="col" className="py-2 text-right font-medium">{currency ? "Amount" : "Count"}</th></tr></thead><tbody>{data.map((point) => <tr key={point.date} className="border-t border-border"><th scope="row" className="py-2 font-normal">{point.date}</th><td className="py-2 text-right">{format(point.value)}</td></tr>)}</tbody></table></div></details>
    </figure>
  );
}
