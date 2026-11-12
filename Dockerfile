# CodexSecure API — production image (Coolify: Build Pack = Dockerfile, port 8084)

# ---- Build stage ----
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Dependencies first for layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Source (see .dockerignore: tests, docs, binaries and secrets are excluded)
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-buildvcs=false go build \
    -trimpath \
    -ldflags='-w -s' \
    -o /build/codexsecure \
    ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /build/codexsecure /app/codexsecure

USER appuser

# Container defaults; all can be overridden by Coolify environment variables.
ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=8084 \
    ENV=production \
    LOG_FORMAT=json \
    LOG_LEVEL=info

EXPOSE 8084

# /api/v1/status is always mounted (no ENABLE_WEB_ROUTES needed) and needs no auth.
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${SERVER_PORT}/api/v1/status" || exit 1

CMD ["/app/codexsecure"]
