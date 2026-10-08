export default function AdminLoading() {
  return <main className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 sm:py-10"><div className="h-7 w-52 animate-pulse rounded bg-border" /><div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{Array.from({ length: 4 }, (_, index) => <div key={index} className="h-28 animate-pulse rounded-2xl border border-border bg-surface" />)}</div></main>;
}
