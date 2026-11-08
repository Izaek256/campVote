import { existsSync } from "node:fs";
import path from "node:path";
import type { NextConfig } from "next";

// Next reads .env only from this directory, but the repository keeps one
// config file at the root, so load that too. Values already in the environment
// win, so apps/web/.env.local, shell exports and the Docker build arg still
// override it.
const rootEnv = path.join(__dirname, "../../.env");
if (existsSync(rootEnv)) {
  process.loadEnvFile(rootEnv);
}

// Deployment-only: the API base to proxy /api and /files to (for example
// http://194.36.88.77:8080). A browser on this origin then talks to one host,
// so CORS does not apply and the API's SameSite=Lax cookies stay first-party.
// A Vercel build defaults to this project's API so a missing variable cannot
// ship an /api route that 404s; API_PROXY_TARGET still overrides it. Unset
// locally and in Docker, where the browser calls the API directly.
const apiProxyTarget = (
  process.env.API_PROXY_TARGET ||
  (process.env.VERCEL ? "http://194.36.88.77:8080" : "")
).replace(/\/$/, "");

// An empty NEXT_PUBLIC_API_URL means same-origin, which needs the proxy. Without
// either, every request lands on web routes that do not exist and 404s.
if (
  process.env.NODE_ENV === "production" &&
  !process.env.NEXT_PUBLIC_API_URL &&
  !apiProxyTarget
) {
  console.warn(
    "Neither NEXT_PUBLIC_API_URL nor API_PROXY_TARGET is set: this build will call " +
      "this origin's /api, where no API is mounted. Set API_PROXY_TARGET in the deploy " +
      "environment (for Vercel: Project → Settings → Environment Variables) and rebuild."
  );
}

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Standalone mode bundles server.js and the minimal node_modules it needs,
  // so the Docker runner stage needs no npm install.
  output: "standalone",
  // In a monorepo the trace must cover the whole repo so shared packages
  // (types, utilities) end up in the standalone output.
  outputFileTracingRoot: path.join(__dirname, "../../"),
  rewrites: async () =>
    apiProxyTarget
      ? [
          { source: "/api/:path*", destination: `${apiProxyTarget}/api/:path*` },
          { source: "/files/:path*", destination: `${apiProxyTarget}/files/:path*` },
        ]
      : [],
};

export default nextConfig;
