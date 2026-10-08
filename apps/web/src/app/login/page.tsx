import { redirect } from "next/navigation";
import { SignInForm } from "@/components/auth-forms";
import { getCurrentUser } from "@/lib/api";
import { authDestination, authHref } from "@/lib/navigation";

export const metadata = {
  title: "Sign in · Stream",
};

export default async function LoginPage({ searchParams }: PageProps<"/login">) {
  const destination = authDestination((await searchParams).next);
  if (await getCurrentUser()) {
    redirect(destination);
  }

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col px-4 py-8">
      <h1 className="text-xl font-semibold tracking-tight">Sign in</h1>
      <p className="mt-2 mb-6 text-sm text-muted">
        {destination.startsWith("/advertiser") ? "Sign in to continue to your advertiser workspace." : "Welcome back. Sign in to share with your community."}
      </p>
      <SignInForm destination={destination} nextHref={authHref("signup", destination)} />
    </main>
  );
}
