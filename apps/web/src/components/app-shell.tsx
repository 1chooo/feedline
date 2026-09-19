import { AccountPanel } from "@/components/account-panel";
import { AuthPanel } from "@/components/auth-panel";
import { Header } from "@/components/header";
import { Sidebar } from "@/components/sidebar";
import type { User } from "@/types/social";

export function AppShell({
  children,
  user,
}: {
  children: React.ReactNode;
  user: User | null;
}) {
  return (
    <div className="flex min-h-full flex-1 flex-col lg:grid lg:grid-cols-[minmax(0,1fr)_32rem_minmax(0,1fr)]">
      <div className="hidden lg:flex">
        <Sidebar username={user?.username} />
      </div>
      <div className="flex min-w-0 flex-1 flex-col lg:border-x lg:border-border">
        <Header user={user} />
        {children}
      </div>
      <div className="hidden lg:block">
        {user ? <AccountPanel user={user} /> : <AuthPanel />}
      </div>
    </div>
  );
}
