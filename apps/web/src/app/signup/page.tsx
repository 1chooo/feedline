import { redirect } from "next/navigation";
import { SignUpForm } from "@/components/auth-forms";
import { getCurrentUser } from "@/lib/api";
import { authDestination, authHref } from "@/lib/navigation";

export const metadata = {
  title: "Sign up · Stream",
};

export default async function SignupPage({ searchParams }: PageProps<"/signup">) {
  const destination = authDestination((await searchParams).next);
  if (await getCurrentUser()) {
    redirect(destination);
  }

  return (
    <main className="mx-auto flex w-full max-w-lg flex-1 flex-col px-4 py-8">
      <h1 className="text-xl font-semibold tracking-tight">{destination.startsWith("/advertiser") ? "Create your advertiser account" : "Create an account"}</h1>
      <p className="mt-2 mb-6 text-sm text-muted">
        {destination.startsWith("/advertiser") ? "Set up your Stream account, then activate a company workspace to plan your first campaign." : "Join Stream to share posts with your community."}
      </p>
      <SignUpForm destination={destination} nextHref={authHref("login", destination)} />
    </main>
  );
}
