# LLM Inference Proxy

An OpenAI-compatible reverse proxy written in Go that routes chat completion requests across multiple backend LLM providers. It features manual dependency injection, API key authentication, Redis-backed sliding-window rate limiting, and Prometheus-based telemetry.

---

## 🗺️ Project Status & Implementation Roadmap

This project is built using Ardan Labs' package-oriented design philosophy. Below is the interactive checklist of what has been scaffolded and what remains to be implemented.

### Phase 1: Project Scaffolding & Architecture
- [x] Establish package-oriented directory layout (`/cmd`, `/internal`, `/pkg`, `/configs`)
- [x] Implement manual dependency injection wiring in `cmd/api/main.go`
- [x] Set up structured JSON logging using `zerolog`
- [x] Design middleware composition helper (`internal/middleware/chain.go`)
- [x] Define Prometheus metrics and register collectors (`RequestDuration`, `RequestTotal`, `UpstreamRetries`)
- [x] Create multi-stage production-ready `Dockerfile`
- [x] Configure local development stack via `docker-compose.yml` (App, Redis, Prometheus, Grafana)
- [x] Create basic Kubernetes manifests (`k8s/`)
- [x] Set up `Makefile` command suite

### Phase 2: Configuration & Authentication
- [x] Parse server ports, timeouts, and API keys from environment
- [x] Parse Bearer Token authorization header
- [x] Validate client requests against configured API keys and propagate keys via context
- [ ] Implement numbered environment variable parsing for upstreams in `loadBackends()` (e.g. `BACKEND_N_NAME`, `BACKEND_N_URL`)

### Phase 3: Proxy Request Handling & Routing
- [x] Define OpenAI-compatible Chat Completion API request/response models
- [x] Implement routing table and model-to-backend indexing
- [x] Implement round-robin routing logic across backends supporting a given model
- [x] Validate required JSON payload parameters (`model`, `messages`)
- [ ] Complete target request construction and header/context forwarding in `/v1/chat/completions`
- [ ] Integrate response forwarding from upstream client to client writer

### Phase 4: Resiliency & Streaming
- [x] Implement jittered exponential backoff algorithm (`pkg/retry/backoff.go`)
- [x] Add Prometheus metric incrementing on upstream retries
- [ ] Implement request body buffering/cloning to enable safe retries on network failures
- [ ] Support Server-Sent Events (SSE) streaming by copying chunks with `http.Flusher`

### Phase 5: Rate Limiting & Telemetry
- [x] Define rate limiting contract interface (`internal/ratelimit/limiter.go`)
- [x] Verify Redis connectivity on startup with health check ping
- [ ] Implement Redis sorted-set sliding window pipeline (ZREMRANGEBYSCORE, ZADD, ZCARD, EXPIRE)
- [ ] Extract validated API key from context and enforce rate limiting in rate-limit middleware
- [ ] Propagate proxy backend/model labels through request context for accurate Prometheus metrics

---

## 🛠️ Getting Started

### Prerequisites
- Go 1.23 or later
- Docker & Docker Compose (optional, for running dependencies locally)

### Local Development Commands

Use the `Makefile` command suite to run, test, and manage the stack:

```bash
# Build the binary to bin/llm-proxy
make build

# Run unit and race tests
make test

# Format all Go source files
make fmt

# Check environment for potential code issues
make vet

# Start the full Docker Compose stack (App, Redis, Prometheus, Grafana)
make docker-up

# Stop and tear down the Docker Compose stack
make docker-down
```

---

## ⚙️ Configuration

The application is configured entirely via environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Port the proxy server binds to | `8080` |
| `READ_TIMEOUT_SEC` | HTTP server read timeout (seconds) | `5` |
| `WRITE_TIMEOUT_SEC` | HTTP server write timeout (seconds) | `60` |
| `IDLE_TIMEOUT_SEC` | HTTP server idle timeout (seconds) | `120` |
| `UPSTREAM_TIMEOUT_SEC` | Upstream HTTP client timeout (seconds) | `30` |
| `MAX_RETRIES` | Max retries for failed upstream requests | `3` |
| `REDIS_URL` | Redis URL connection string | *Required* |
| `RATE_LIMIT_WINDOW_SEC` | Duration of rate limiting window (seconds) | `60` |
| `RATE_LIMIT_MAX_REQUESTS` | Allowed requests per key per window | `100` |
| `API_KEYS` | Comma-separated list of valid client keys | *Required* |
