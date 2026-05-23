# Quick Reference

## Run

```bash
make stack                          # Start collector + Tempo + Loki + Prometheus + Grafana
make server                         # Run the example server (HTTP :8080)
# or: make run                      # both of the above
```

## Test

```bash
curl "http://localhost:8080/hello?name=World"
curl "http://localhost:8080/health"
open http://localhost:3000          # Grafana (admin/admin) → Explore
```

## Copy the template

```bash
cp -r pkg/telemetry /path/to/your-service/pkg/telemetry
```

## Initialize telemetry

```go
shutdown, err := telemetry.Setup(ctx, telemetry.Config{
	ServiceName:    "my-service",
	ServiceVersion: "1.0.0",
	Environment:    "production",
})
if err != nil {
	log.Fatal(err)
}
defer shutdown(context.Background())

// Route logs through OpenTelemetry.
slog.SetDefault(slog.New(otelslog.NewHandler("my-service")))
```

## Instrument an HTTP server

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// One server span + http.server.* metrics per request, automatically.
handler := otelhttp.NewHandler(mux, "GET /route")
```

## Optional: custom instrumentation

The example app relies entirely on `otelhttp` and adds no manual spans/metrics.
The snippets below are for when you want to instrument your own logic.

### Create spans

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(ctx, "operation",
	trace.WithAttributes(attribute.String("user.id", "123")))
defer span.End()

span.AddEvent("processing_done")          // event
span.RecordError(err)                      // error
span.SetStatus(codes.Error, err.Error())   // status
```

### Metrics

```go
meter := otel.Meter("my-service")

counter, _ := meter.Int64Counter("requests.total",
	metric.WithDescription("Total requests"))
counter.Add(ctx, 1, metric.WithAttributes(attribute.String("route", "/hello")))

hist, _ := meter.Float64Histogram("request.duration", metric.WithUnit("s"))
hist.Record(ctx, 0.142)
```

### Logs

```go
// With slog routed through otelslog, logs are correlated with the active span.
slog.InfoContext(ctx, "served request", "route", "/hello")
```

## Environment variables

All exporter config is read from the environment (no code changes needed):

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318   # local collector (http:// = plaintext)
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf           # default; or "grpc"
OTEL_EXPORTER_OTLP_HEADERS=api-key=YOUR_KEY         # hosted backend auth
OTEL_TRACES_SAMPLER=parentbased_traceidratio        # sampling strategy
OTEL_TRACES_SAMPLER_ARG=0.1                          # ...sample 10% of traces
OTEL_METRICS_EXPORTER=none                           # disable a signal entirely
OTEL_RESOURCE_ATTRIBUTES=team=payments               # extra resource attributes
```

## Make targets

```
make help        Show all targets
make stack       Start observability stack
make server      Run the example server
make run         Stack + server
make build       Build into bin/
make test        Run tests
make lint        Vet + staticcheck
make tidy        Tidy go.mod / go.sum
make stack-down  Stop the stack
make clean       Remove build artifacts
```
