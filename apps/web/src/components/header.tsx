import Link from "next/link";

export function Header() {
  return (
    <header className="sticky top-0 z-10 border-b border-border bg-surface/90 backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-lg items-center justify-between px-4">
        <Link href="/" className="text-base font-semibold tracking-tight">
          Stream
        </Link>
        <button
          type="button"
          className="rounded-full bg-foreground px-3.5 py-1.5 text-sm font-medium text-background"
        >
          Sign in
        </button>
      </div>
    </header>
  );
}
