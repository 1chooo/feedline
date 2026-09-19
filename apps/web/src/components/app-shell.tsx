import { Header } from "@/components/header";
import { Sidebar } from "@/components/sidebar";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-full flex-1">
      <Sidebar />
      <div className="flex min-w-0 w-full flex-1 flex-col lg:max-w-lg lg:border-x lg:border-border">
        <Header />
        {children}
      </div>
    </div>
  );
}
