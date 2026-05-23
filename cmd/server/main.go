// Command server is a small HTTP service that demonstrates how to wire the
// telemetry package into an application: initialize telemetry, route logs
// through slog, serve instrumented handlers, and shut everything down cleanly.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"

	"github.com/otel-golang/internal/api"
	"github.com/otel-golang/pkg/telemetry"
)

const (
	serviceName    = "greeter-service"
	serviceVersion = "1.0.0"
	addr           = ":8080"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Cancel the root context on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    "development",
	})
	if err != nil {
		return err
	}
	defer func() {
		// Flush telemetry with a fresh, bounded context — ctx is already canceled.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}()

	// Route the standard library's slog through OpenTelemetry so application logs
	// are correlated with traces and exported to the collector.
	slog.SetDefault(slog.New(otelslog.NewHandler(serviceName)))

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.NewHandler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
