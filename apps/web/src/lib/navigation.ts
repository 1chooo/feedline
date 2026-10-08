// Authentication destinations are limited to product routes. Query values are
// rebuilt rather than forwarding an arbitrary URL into a server redirect.
export function authDestination(value: unknown): string {
  if (typeof value !== "string" || !value.startsWith("/") || value.startsWith("//")) return "/";
  try {
    const url = new URL(value, "https://stream.invalid");
    if (url.origin !== "https://stream.invalid") return "/";
    if (!["/", "/advertiser", "/admin", "/settings"].includes(url.pathname)) return "/";
    const packageID = selectedPackageID(url.searchParams.get("package"));
    return url.pathname === "/advertiser" && packageID ? `/advertiser?package=${packageID}` : url.pathname;
  } catch {
    return "/";
  }
}

export function selectedPackageID(value: unknown): number | undefined {
  if (typeof value !== "string" || !/^[1-9]\d*$/.test(value)) return undefined;
  const number = Number(value);
  return Number.isSafeInteger(number) ? number : undefined;
}

export function authHref(kind: "login" | "signup", destination: string) {
  return destination === "/" ? `/${kind}` : `/${kind}?next=${encodeURIComponent(authDestination(destination))}`;
}
