export type DateRange = { from: string; to: string; error?: string };

export function reportingRange(from: unknown, to: unknown, now = new Date()): DateRange {
  const today = now.toISOString().slice(0, 10);
  const defaultFrom = new Date(`${today}T00:00:00Z`);
  defaultFrom.setUTCDate(defaultFrom.getUTCDate() - 29);
  const fallback = { from: defaultFrom.toISOString().slice(0, 10), to: today };
  const end = typeof to === "string" && to ? to : today;
  const endDate = validDate(end);
  if (!endDate) return { ...fallback, error: "Enter valid reporting dates in YYYY-MM-DD format." };
  const automaticStart = new Date(endDate);
  automaticStart.setUTCDate(automaticStart.getUTCDate() - 29);
  const start = typeof from === "string" && from ? from : automaticStart.toISOString().slice(0, 10);
  const startDate = validDate(start);
  if (!startDate) return { ...fallback, error: "Enter valid reporting dates in YYYY-MM-DD format." };
  if (endDate < startDate) return { from: start, to: end, error: "The end date must be on or after the start date." };
  if ((endDate.valueOf() - startDate.valueOf()) / 86400000 > 89) return { from: start, to: end, error: "Choose a reporting period of 90 days or less." };
  return { from: start, to: end };
}

function validDate(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return null;
  const date = new Date(`${value}T00:00:00Z`);
  return Number.isFinite(date.valueOf()) && date.toISOString().slice(0, 10) === value ? date : null;
}
