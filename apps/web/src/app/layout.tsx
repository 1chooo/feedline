import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { AppShell } from "@/components/app-shell";
import { ThemeScript } from "@/components/theme-script";
import { getCurrentUser } from "@/lib/api";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Stream",
  description: "A simple public feed of ads as posts.",
};

export default async function RootLayout({ children }: LayoutProps<"/">) {
  const user = await getCurrentUser();

  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col bg-background font-sans text-foreground">
        <a href="#main-content" className="fixed left-4 top-3 z-50 -translate-y-24 rounded-xl bg-foreground px-4 py-3 text-sm font-medium text-background focus:translate-y-0">Skip to content</a>
        <ThemeScript />
        <AppShell user={user}>{children}</AppShell>
      </body>
    </html>
  );
}
