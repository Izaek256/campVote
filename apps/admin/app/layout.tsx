import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "campVote Admin — MMU campus elections",
  description: "Administration interface for Mountains of the Moon University campus elections.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
