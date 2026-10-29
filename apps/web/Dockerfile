# Multi-stage build for Next.js web app
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files from the root (build context is the repository root)
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json ./apps/web/

# Install pnpm
RUN npm install -g pnpm@10.33.4

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the application
RUN pnpm build --filter=@campvote/web

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

# Copy built files
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next /app/.next
COPY --chown=nextjs:nodejs apps/web/package.json apps/web/next.config.ts ./

# Copy public directory if it exists
RUN mkdir -p ./public && \
    if [ -d "/app/apps/web/public" ]; then \
        cp -r /app/apps/web/public/* ./public/; \
    fi

# Set environment to production
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Expose port
EXPOSE 3000

# Start the server
USER nextjs
CMD ["npm", "run", "start"]
