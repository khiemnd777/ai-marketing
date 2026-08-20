import type { Metadata } from "next";
import type { ReactNode } from "react";
import { QueryProvider } from "@/components/query-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "AI Product Marketing Studio", template: "%s · AI Studio" },
  description: "Nền tảng vận hành nội bộ cho nội dung, video và Meta Ads dựa trên Product Truth.",
  robots: { index: false, follow: false },
};

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="vi">
      <body><QueryProvider>{children}</QueryProvider></body>
    </html>
  );
}
