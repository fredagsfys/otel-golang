COMPOSE := docker compose -f observability/docker-compose.yaml

.PHONY: help build run server stack stack-down test lint tidy clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the example server into bin/
	go build -o bin/server ./cmd/server

run: stack server ## Start the observability stack and run the server

# OTEL_EXPORTER_OTLP_ENDPOINT carries an explicit http:// scheme so the exporter
# talks plaintext to the local collector (the OTLP default is TLS). The 5s metric
# interval just gives faster feedback locally (the OTEL default is 60s).
OTEL_DEV_ENV := OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 OTEL_METRIC_EXPORT_INTERVAL=5000

server: ## Run the example server (expects the stack to be up)
	$(OTEL_DEV_ENV) go run ./cmd/server

stack: ## Start the supporting observability stack (collector, Tempo, Loki, Prometheus, Grafana)
	$(COMPOSE) up -d

stack-down: ## Stop the observability stack
	$(COMPOSE) down

test: ## Run tests
	go test ./...

lint: ## Vet + staticcheck (downloaded on demand via go run)
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...

tidy: ## Tidy go.mod / go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -rf bin/
	go clean
