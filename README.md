# otel-golang

A clean, idiomatic Go template for **OpenTelemetry** — traces, metrics, and logs
exported over vendor-neutral OTLP. Drop the [`pkg/telemetry`](pkg/telemetry/telemetry.go)
package into your service, call `telemetry.Setup` once, and point it at any
OTLP-compatible backend (Tempo, Datadog, Honeycomb, New Relic, …) — **no code
changes to switch backends**.

The example HTTP service in [`cmd/server`](cmd/server/main.go) shows the
recommended setup with **zero hand-written instrumentation** — `otelhttp` does
it. Everything under [`observability/`](observability/) is a local observability stack used
only to *see* the telemetry while you develop; it is supporting infrastructure,
not part of the template you ship.

## What's inside

| Path | Role |
| --- | --- |
| [`pkg/telemetry/`](pkg/telemetry/telemetry.go) | **The template.** Reusable OTLP setup for traces, metrics, and logs. Copy this into your service. |
| [`cmd/server/`](cmd/server/main.go) | Example service wiring telemetry into an HTTP server. |
| [`internal/api/`](internal/api/handlers.go) | Example handlers — plain funcs wrapped in `otelhttp` for fully automatic spans + metrics. |
| [`observability/`](observability/) | Supporting stack to test against: OTel Collector, Tempo, Loki, Prometheus, Grafana. |

> The example service is organized by transport (`internal/api`) because it's a
> demo. As your service grows real domains, slice `internal/` by domain instead —
> e.g. `internal/orders/` holding that domain's handler, logic, and storage
> together — rather than splitting by technical layer.

## Quick start

```bash
# 1. Start the supporting observability stack (collector + Tempo/Loki/Prometheus/Grafana)
make stack

# 2. Run the example server (HTTP :8080)
make server

# 3. Generate some telemetry
curl "http://localhost:8080/hello?name=World"
curl "http://localhost:8080/health"
```

`make run` does steps 1 and 2 together. Run `make help` to list all targets.

### See the telemetry

Open Grafana at <http://localhost:3000> (login `admin` / `admin`) → **Explore**:

- **Traces** — datasource `Tempo`, run TraceQL `{ resource.service.name = "greeter-service-golang" }`
- **Metrics** — datasource `Prometheus`, search for `http_server_request_duration_seconds_count`
- **Logs** — datasource `Loki`, query `{service_name="greeter-service-golang"}`

Other endpoints: Prometheus <http://localhost:9090>, Tempo API <http://localhost:3200>,
collector health <http://localhost:13133>. To watch what the collector receives:

```bash
docker compose -f observability/docker-compose.yaml logs -f otel-collector
```

## Using the template in your service

Copy [`pkg/telemetry/`](pkg/telemetry/telemetry.go) into your module, then:

```go
package main

import (
	"context"
	"log"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"github.com/your-org/your-svc/pkg/telemetry"
)

func main() {
	ctx := context.Background()

	shutdown, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		Environment:    "production",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(context.Background())

	// Route slog through OpenTelemetry so logs correlate with traces.
	slog.SetDefault(slog.New(otelslog.NewHandler("my-service")))

	// ... your app ...
}
```

`Setup` initializes the global tracer, meter, and logger providers plus context
propagation, and returns a single `shutdown` func that flushes everything.

> Prefer to start from this whole repo? Rename the module in `go.mod`
> (`go mod edit -module github.com/your-org/your-svc`) and update the matching
> imports — that's the only project-specific wiring.

### Instrumenting code

HTTP servers and clients are instrumented automatically with `otelhttp` — the
example handlers carry **no manual span or metric code**. When you want to
instrument your own business logic, pull a tracer/meter from the global providers
that `Setup` registered:

```go
tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(ctx, "doWork",
	trace.WithAttributes(attribute.String("user.id", "123")))
defer span.End()

if err := work(ctx); err != nil {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
```

See [QUICK_REFERENCE.md](QUICK_REFERENCE.md) for span, metric, and log snippets.

## Configuration

`telemetry.Config` only carries the service's identity. Everything else —
transport, endpoint, auth, sampling — is read from the standard `OTEL_*`
environment variables (via [`autoexport`](https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport)),
so the same binary is configured purely through its environment:

| Environment variable | Purpose | Default |
| --- | --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector / backend address | `localhost:4318` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` or `grpc` | `http/protobuf` |
| `OTEL_EXPORTER_OTLP_HEADERS` | Auth headers for hosted backends | _(none)_ |
| `OTEL_TRACES_SAMPLER` / `_ARG` | Sampling strategy / ratio | `parentbased_always_on` |
| `OTEL_{TRACES,METRICS,LOGS}_EXPORTER` | Set to `none` to disable a signal | `otlp` |
| `OTEL_METRIC_EXPORT_INTERVAL` | Metric export interval (ms) | `60000` |
| `OTEL_RESOURCE_ATTRIBUTES` / `OTEL_SERVICE_NAME` | Extra resource attributes | _(none)_ |

> **Talking to the local (plaintext) collector:** OTLP exporters default to a
> *TLS* connection. To reach the plaintext collector in `observability/`, give the
> endpoint an explicit `http://` scheme:
> `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318`. The `make server` target
> sets this (and a faster 5s metric interval) for you. For gRPC instead, use
> `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` with `http://localhost:4317`. Against a real
> hosted backend you'd use its `https://` endpoint, which needs no special flag.

### Switching backends

Point the endpoint (and headers, for auth) at any OTLP backend:

```bash
# Honeycomb (OTLP/HTTP, the default protocol)
export OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io
export OTEL_EXPORTER_OTLP_HEADERS=x-honeycomb-team=YOUR_KEY

# Grafana Cloud / Tempo over gRPC
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
export OTEL_EXPORTER_OTLP_ENDPOINT=tempo.example.com:4317
```

> **Want truly zero-code instrumentation?** OpenTelemetry's eBPF
> [auto-instrumentation](https://github.com/open-telemetry/opentelemetry-go-instrumentation)
> attaches to a running Go binary with no source changes, but it runs as a
> privileged, Linux-only sidecar process. This template instruments in-process
> instead — portable, no special privileges, and explicit about what's traced.

## Make targets

```
make help        Show all targets
make stack       Start the supporting observability stack
make server      Run the example server
make run         Start the stack and run the server
make build       Build the server into bin/
make test        Run tests
make lint        Vet + staticcheck
make tidy        Tidy go.mod / go.sum
make stack-down  Stop the observability stack
make clean       Remove build artifacts
```

## License

MIT
