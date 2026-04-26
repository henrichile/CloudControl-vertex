# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Cache de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Build del binario
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o cloudcontrol ./cmd/server

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /build/cloudcontrol .

RUN mkdir -p /data && chown -R appuser:appgroup /app /data

USER appuser

EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENV DATABASE_PATH=/data/cloudcontrol.db \
    PORT=8080

ENTRYPOINT ["./cloudcontrol"]
