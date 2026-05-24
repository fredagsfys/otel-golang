// Package telemetry wires up OpenTelemetry tracing, metrics, and logging.
//
// Exporters are built from the standard OTEL_* environment variables via
// autoexport, so switching transport (grpc/http), endpoint, headers, or backend
// is pure configuration — no code changes. Copy this package into your service
// and call Setup once at startup.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// Config identifies the service producing telemetry. Everything else — OTLP
// endpoint, protocol, headers, trace sampling, metric interval — is read from
// the standard OTEL_* environment variables.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
}

// Setup initializes the global tracer, meter, and logger providers plus context
// propagation, and returns a shutdown function that flushes and closes them all.
// Call shutdown with a bounded context before the process exits.
//
// Relevant environment variables (all optional):
//
//	OTEL_EXPORTER_OTLP_ENDPOINT   collector/backend address (default http://localhost:4318)
//	OTEL_EXPORTER_OTLP_PROTOCOL   "http/protobuf" (default) or "grpc"
//	OTEL_EXPORTER_OTLP_HEADERS    auth headers for hosted backends, e.g. "api-key=..."
//	OTEL_TRACES_SAMPLER[_ARG]     sampling strategy (default parentbased_always_on)
//	OTEL_{TRACES,METRICS,LOGS}_EXPORTER  set to "none" to disable a signal
//	OTEL_SERVICE_NAME             overrides cfg.ServiceName
//	OTEL_RESOURCE_ATTRIBUTES      extra/override resource attributes, e.g. "team=payments"
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	// Surface SDK-internal errors (export failures, etc.) instead of swallowing them.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("telemetry: %v", err)
	}))

	// Propagate trace context and baggage across service boundaries.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	// Collect provider shutdowns; on any init error, roll back what already started.
	var shutdowns []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var errs error
		for _, fn := range shutdowns {
			errs = errors.Join(errs, fn(ctx))
		}
		return errs
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, shutdown(ctx))
		}
	}()

	// Traces. The sampler is taken from OTEL_TRACES_SAMPLER[_ARG] automatically.
	spanExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: span exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
	)
	shutdowns = append(shutdowns, tp.Shutdown)
	otel.SetTracerProvider(tp)

	// Metrics. NewMetricReader returns a ready periodic reader for OTLP.
	metricReader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: metric reader: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	shutdowns = append(shutdowns, mp.Shutdown)
	otel.SetMeterProvider(mp)

	// Logs. Registered globally so the otelslog bridge picks it up.
	logExporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	shutdowns = append(shutdowns, lp.Shutdown)
	global.SetLoggerProvider(lp)

	return shutdown, nil
}

// newResource builds the resource describing this service. The cfg fields act as
// defaults that the standard OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES env
// vars may override.
//
// Order matters: resource Merge is last-value-wins, so WithFromEnv() is applied
// AFTER WithAttributes() to let env vars override the code defaults (idiomatic
// OTel precedence: code = defaults, env = override). When OTEL_SERVICE_NAME /
// OTEL_RESOURCE_ATTRIBUTES are unset, the env detector contributes nothing for
// those keys, so the cfg defaults still hold.
func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
		resource.WithFromEnv(), // OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES override the above
	)
}
