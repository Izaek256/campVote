# Multi-stage build for Go API
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files from apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (static binary, no external dependencies needed)
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./apps/api/cmd/server

# Final stage - use scratch for minimal image (static binary needs no runtime)
FROM scratch

WORKDIR /app

COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the server
CMD ["/app/server"]
