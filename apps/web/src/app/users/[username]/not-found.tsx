import Link from "next/link";

export default function UserNotFound() {
  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col items-center justify-center px-4 py-16 text-center">
      <h1 className="text-xl font-semibold">User not found</h1>
      <p className="mt-2 text-sm text-muted">
        That username is not in the mock feed.
      </p>
      <Link
        href="/"
        className="mt-6 text-sm font-medium underline-offset-2 hover:underline"
      >
        Back to feed
      </Link>
    </main>
  );
}
