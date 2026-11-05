# ── Builder ────────────────────────────────────────────────────────────────────
# Build context is the repository root so pnpm can resolve the workspace.
FROM node:20-alpine AS builder

WORKDIR /app

# Install pnpm (pinned version from packageManager field)
RUN npm install -g pnpm@10.33.4

# Copy workspace manifests first for better layer caching
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY apps/web/package.json ./apps/web/

# Install all workspace deps (frozen so CI-parity is maintained)
RUN pnpm install --frozen-lockfile

# Build-time public API URL. Baked into the JS bundle, so it must be the URL
# a *browser* can reach — not an internal Docker hostname.
# Default points at localhost for local development.
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_TELEMETRY_DISABLED=1

# Copy source and build
COPY apps/web ./apps/web
COPY biome.jsonc ./

RUN pnpm --filter @campvote/web build

# ── Runtime ────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS runner

WORKDIR /app

RUN addgroup -g 1001 -S nodejs && \
    adduser  -S -u 1001 -G nodejs nextjs

# next.config.ts output: "standalone" generates a self-contained server.
# The standalone directory already contains its own node_modules copy.
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/static     ./apps/web/.next/static

# Public directory (favicon, images, etc.)
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/public           ./apps/web/public

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

USER nextjs
EXPOSE 3000

# The standalone server entry point produced by Next.js
CMD ["node", "apps/web/server.js"]
