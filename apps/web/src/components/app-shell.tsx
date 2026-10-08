"use client";

import { usePathname } from "next/navigation";
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
	const pathname = usePathname();
	if (pathname === "/advertise") {
		return <div id="main-content" tabIndex={-1}>{children}</div>;
	}
	const workspace = pathname.startsWith("/admin") || pathname.startsWith("/advertiser");
  const authentication = pathname === "/login" || pathname === "/signup";
  return (
    <div className={`mx-auto flex min-h-full w-full flex-1 flex-col lg:grid ${workspace || authentication ? "max-w-[100rem] lg:grid-cols-[14rem_minmax(0,1fr)]" : "max-w-7xl lg:grid-cols-[13rem_minmax(0,36rem)_minmax(0,1fr)]"}`}>
      <div className="hidden lg:flex">
        <Sidebar user={user} />
      </div>
      <div id="main-content" tabIndex={-1} className="flex min-w-0 flex-1 flex-col lg:border-x lg:border-border">
        <Header user={user} />
        {children}
      </div>
      {!workspace && !authentication ? <div className="hidden lg:block">
        {user ? <AccountPanel user={user} /> : <AuthPanel />}
      </div> : null}
    </div>
  );
}
