package telemetry_test

import (
	"context"
	"testing"

	"github.com/otel-golang/pkg/telemetry"
)

// TestSetup verifies that Setup wires up all three signals and returns a working
// shutdown. The OTEL_*_EXPORTER=none settings make autoexport build no-op
// exporters, so the test is hermetic — no collector or network required.
func TestSetup(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	t.Setenv("OTEL_LOGS_EXPORTER", "none")

	shutdown, err := telemetry.Setup(context.Background(), telemetry.Config{
		ServiceName:    "test-service",
		ServiceVersion: "0.0.0",
		Environment:    "test",
	})
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Setup returned a nil shutdown function")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}
}
