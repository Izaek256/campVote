import type { NextConfig } from "next";
import path from "node:path";

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
