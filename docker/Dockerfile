# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Stage 1 — builder
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

# Install ca-certificates so the binary can make TLS calls to upstream APIs.
RUN apk add --no-cache ca-certificates git

WORKDIR /build

# Cache dependency downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a statically-linked binary with debug info stripped.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-s -w" \
      -trimpath \
      -o /build/llm-proxy \
      ./cmd/api

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2 — final (distroless)
# ─────────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

# Copy TLS certificates from builder so upstream HTTPS calls succeed.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /build/llm-proxy /llm-proxy

# Run as the nonroot user provided by distroless.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/llm-proxy"]
