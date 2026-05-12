import type { Metadata } from "next";

import "./globals.css";

export const metadata: Metadata = {
  title: "Kick Logs",
  description: "Self-hosted Kick chat log search",
  icons: {
    icon: "/app-logo.png",
    shortcut: "/app-logo.png",
    apple: "/app-logo.png"
  }
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="tr" className="dark">
      <body>{children}</body>
    </html>
  );
}
