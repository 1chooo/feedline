import { ThemeSetting } from "@/components/theme-setting";

export const metadata = {
  title: "Settings · Stream",
};

export default function SettingsPage() {
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
    </main>
  );
}
