import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/components/auth-provider";
import { ToastProvider } from "@/components/toast";
import { CAMPUS_NAME } from "@/lib/config";

export const metadata: Metadata = {
  title: `campVote — ${CAMPUS_NAME} elections`,
  description: `Vote in authorised ${CAMPUS_NAME} elections.`,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=Space+Grotesk:wght@400;500;600;700&display=swap"
        />
        {/* Apply the saved theme before paint so every page starts in the
            chosen mode instead of flashing the default. */}
        <script
          dangerouslySetInnerHTML={{
            __html:
              'try{var t=localStorage.getItem("theme");if(t&&t!=="system"){document.documentElement.setAttribute("data-theme",t);}}catch(e){}',
          }}
        />
      </head>
      <body>
        <AuthProvider>
          <ToastProvider>{children}</ToastProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
