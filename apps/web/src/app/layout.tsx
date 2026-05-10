import type { Metadata } from "next";

import "./globals.css";

export const metadata: Metadata = {
  title: "Kick Logs",
  description: "Kick Logs frontend shell"
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
