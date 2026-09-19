export function AuthPanel() {
  return (
    <aside className="sticky top-0 h-svh w-full max-w-80 px-6 py-10">
      <h2 className="text-xl font-semibold tracking-tight">
        Log in or sign up for Stream
      </h2>
      <p className="mt-2 text-sm leading-6 text-muted">
        You can scroll the feed now. Sign in later to post.
      </p>
      <button
        type="button"
        className="mt-6 w-full rounded-full bg-foreground py-2.5 text-sm font-medium text-background"
      >
        Sign in
      </button>
      <button
        type="button"
        className="mt-3 w-full rounded-full border border-border py-2.5 text-sm font-medium"
      >
        Sign up
      </button>
    </aside>
  );
}
