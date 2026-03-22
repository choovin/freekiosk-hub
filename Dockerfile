# Build stage
FROM docker.1ms.run/library/golang:1.25-alpine AS builder

WORKDIR /build

# Install required tools
RUN apk add --no-cache git make

# Install templ for UI generation
RUN go install github.com/a-h/templ/cmd/templ@v0.3.977

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate templ UI components
RUN templ generate

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/freekiosk-hub ./cmd/server/main.go

# Runtime stage
FROM docker.1ms.run/library/alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS and Tailscale dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D appuser

# Copy binary from builder
COPY --from=builder /build/bin/freekiosk-hub /app/freekiosk-hub

# Copy i18n translation files
COPY --from=builder /build/internal/i18n/locales /app/locales

# Create directories for data and media
RUN mkdir -p /app/data /app/media && \
    chown -R appuser:appgroup /app

USER appuser

# Expose ports
# 8081: Web UI
# 8080: Kiosk API (internal)
EXPOSE 8081 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/health || exit 1

ENTRYPOINT ["/app/freekiosk-hub"]
