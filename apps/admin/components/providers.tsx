"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

/**
 * Wraps the admin app in all client-side providers.
 *
 * Using a component keeps layout.tsx a Server Component — required for Next.js
 * metadata exports to work correctly.
 */
export function Providers({ children }: { children: React.ReactNode }) {
  // Create one QueryClient per session, inside state so it is not shared
  // between different users in server-side rendering contexts.
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 1,
          },
        },
      })
  );

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
