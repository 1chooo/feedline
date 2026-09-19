import { ThemeSetting } from "@/components/theme-setting";
import { SignOutButton } from "@/components/sign-out-button";
import { getCurrentUser } from "@/lib/api";

export const metadata = {
  title: "Settings · Stream",
};

export default async function SettingsPage() {
  const user = await getCurrentUser();

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col px-4 py-6">
      <h1 className="text-xl font-semibold tracking-tight">Settings</h1>
      <section className="mt-6 rounded-2xl border border-border bg-surface p-4">
        <h2 className="text-sm font-medium">Appearance</h2>
        <p className="mt-1 mb-3 text-sm text-muted">
          Choose how Stream looks on this device.
        </p>
        <ThemeSetting />
      </section>
      {user ? (
        <section className="mt-4 rounded-2xl border border-border bg-surface p-4">
          <h2 className="text-sm font-medium">Account</h2>
          <p className="mt-1 mb-3 text-sm text-muted">
            Signed in as @{user.username}.
          </p>
          <SignOutButton className="rounded-full border border-border px-4 py-2 text-sm font-medium" />
        </section>
      ) : null}
    </main>
  );
}
