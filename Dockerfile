# ── Build stage ──
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /invoice-bot ./cmd/bot

# ── Runtime stage ──
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary
COPY --from=builder /invoice-bot .

# Copy default style (can be overridden via volume mount)
COPY defaults/style.yaml /app/defaults/style.yaml

USER appuser

ENTRYPOINT ["./invoice-bot"]
