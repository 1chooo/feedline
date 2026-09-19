import { AuthPanel } from "@/components/auth-panel";
import { Header } from "@/components/header";
import { Sidebar } from "@/components/sidebar";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-full flex-1 flex-col lg:grid lg:grid-cols-[minmax(0,1fr)_32rem_minmax(0,1fr)]">
      <div className="hidden lg:flex">
        <Sidebar />
      </div>
      <div className="flex min-w-0 flex-1 flex-col lg:border-x lg:border-border">
        <Header />
        {children}
      </div>
      <div className="hidden lg:block">
        <AuthPanel />
      </div>
    </div>
  );
}
