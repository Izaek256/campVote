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

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Standalone mode bundles server.js and the minimal node_modules it needs,
  // so the Docker runner stage needs no npm install.
  output: "standalone",
  // In a monorepo the trace must cover the whole repo so shared packages
  // (types, utilities) end up in the standalone output.
  outputFileTracingRoot: path.join(__dirname, "../../"),
};

export default nextConfig;
