# Multi-stage build for Next.js admin app
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/admin/package.json ./apps/admin/

# Install pnpm
RUN npm install -g pnpm@10.33.4

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the application
RUN pnpm build --filter=@campvote/admin

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

# Copy built files
COPY --from=builder --chown=nextjs:nodejs /app/apps/admin/.next /app/.next
COPY --chown=nextjs:nodejs apps/admin/package.json apps/admin/next.config.ts ./
COPY --chown=nextjs:nodejs apps/admin/public ./public

# Set environment to production
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Expose port
EXPOSE 3001

# Start the server
USER nextjs
CMD ["npm", "run", "start"]
