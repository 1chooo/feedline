import { redirect } from "next/navigation";
import { SignUpForm } from "@/components/auth-forms";
import { getCurrentUser } from "@/lib/api";

export const metadata = {
  title: "Sign up · Stream",
};

export default async function SignupPage() {
  if (await getCurrentUser()) {
    redirect("/");
  }

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col px-4 py-8">
      <h1 className="text-xl font-semibold tracking-tight">Create an account</h1>
      <p className="mt-2 mb-6 text-sm text-muted">
        Join Stream to publish posts in the public feed.
      </p>
      <SignUpForm />
    </main>
  );
}
