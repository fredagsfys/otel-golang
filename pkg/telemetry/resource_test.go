package telemetry

import (
	"context"
	"testing"

	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// serviceName extracts the resolved service.name from the resource built for cfg.
func serviceName(t *testing.T, cfg Config) string {
	t.Helper()
	res, err := newResource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("newResource returned error: %v", err)
	}
	v, ok := res.Set().Value(semconv.ServiceNameKey)
	if !ok {
		t.Fatal("resource has no service.name attribute")
	}
	return v.AsString()
}

// TestResourceServiceNameDefault verifies that when OTEL_SERVICE_NAME is unset,
// the hardcoded cfg.ServiceName default holds.
func TestResourceServiceNameDefault(t *testing.T) {
	// t.Setenv guarantees these are unset for this test and restored after.
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")

	if got := serviceName(t, Config{ServiceName: "default-service"}); got != "default-service" {
		t.Errorf("service.name = %q, want %q (cfg default)", got, "default-service")
	}
}

// TestResourceServiceNameEnvOverride verifies that OTEL_SERVICE_NAME overrides
// the hardcoded cfg.ServiceName default (env wins).
func TestResourceServiceNameEnvOverride(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-env")

	if got := serviceName(t, Config{ServiceName: "default-service"}); got != "from-env" {
		t.Errorf("service.name = %q, want %q (OTEL_SERVICE_NAME override)", got, "from-env")
	}
}
