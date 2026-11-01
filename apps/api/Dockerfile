# ── Builder ────────────────────────────────────────────────────────────────────
# golang:alpine is always the latest stable, and GOTOOLCHAIN=auto lets it
# download the exact version named in go.mod (1.26.4).
FROM golang:alpine AS builder

ENV GOTOOLCHAIN=auto
ENV CGO_ENABLED=0
ENV GOOS=linux

# git is needed by some go get operations inside module downloads
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache dependency downloads before copying source, so edits don't re-download.
COPY go.mod go.sum ./
RUN go mod download

# Copy all source (the build context is apps/api/)
COPY . .

# Build all four binaries into /out/
RUN mkdir -p /out && \
    go build -o /out/server  ./cmd/server  && \
    go build -o /out/migrate ./cmd/migrate && \
    go build -o /out/bootstrap ./cmd/bootstrap && \
    go build -o /out/seed   ./cmd/seed

# ── Runtime ────────────────────────────────────────────────────────────────────
FROM alpine:3.20

# CA certs (for outbound HTTPS: S3, SMTP, OAuth) and timezone data (for EAT)
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

# Copy all binaries
COPY --from=builder /out/server    ./server
COPY --from=builder /out/migrate   ./migrate
COPY --from=builder /out/bootstrap ./bootstrap
COPY --from=builder /out/seed      ./seed

# Entrypoint: run migrate, then start the server
COPY entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

USER app

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=5 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
