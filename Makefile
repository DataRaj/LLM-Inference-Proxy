# ─────────────────────────────────────────────────────────────────────────────
# llm-proxy Makefile
# ─────────────────────────────────────────────────────────────────────────────

BINARY      := llm-proxy
CMD         := ./cmd/api
BIN_DIR     := bin
IMAGE       := ghcr.io/dataraj/llm-proxy
TAG         ?= local

.PHONY: all build run test lint vet fmt docker-build docker-up docker-down clean tidy help

all: build

## build: Compile the binary to ./bin/llm-proxy
build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN_DIR)/$(BINARY) $(CMD)
	@echo "✓ built $(BIN_DIR)/$(BINARY)"

## run: Build and run locally (requires env vars to be set)
run: build
	./$(BIN_DIR)/$(BINARY)

## test: Run all tests with race detector
test:
	go test -race -count=1 ./...

## lint: Run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format all Go source files
fmt:
	gofmt -w -s .

## tidy: Tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## docker-build: Build the Docker image using the multi-stage Dockerfile
docker-build:
	docker build -f docker/Dockerfile -t $(IMAGE):$(TAG) .
	@echo "✓ built $(IMAGE):$(TAG)"

## docker-up: Start all services via docker-compose
docker-up:
	docker compose -f docker/docker-compose.yml up --build -d
	@echo "✓ stack started — proxy on http://localhost:8080"

## docker-down: Stop and remove all containers
docker-down:
	docker compose -f docker/docker-compose.yml down

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## help: Print this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'
