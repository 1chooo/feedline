import { redirect } from "next/navigation";
import { SignInForm } from "@/components/auth-forms";
import { getCurrentUser } from "@/lib/api";

export const metadata = {
  title: "Sign in · Stream",
};

export default async function LoginPage() {
  if (await getCurrentUser()) {
    redirect("/");
  }

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col px-4 py-8">
      <h1 className="text-xl font-semibold tracking-tight">Sign in</h1>
      <p className="mt-2 mb-6 text-sm text-muted">
        Use a Stream account to post.
      </p>
      <SignInForm />
    </main>
  );
}
