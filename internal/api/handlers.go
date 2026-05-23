// Package api is the example HTTP service that exercises the telemetry setup.
// Handlers are plain functions: wrapping each route in otelhttp produces a server
// span and the standard http.server.* metrics automatically, with no manual
// instrumentation in the handlers themselves. Logs flow through slog, which is
// routed to OpenTelemetry in main, so they are correlated with the request span.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type helloResponse struct {
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHandler returns the fully instrumented HTTP handler for the example service.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /hello", otelhttp.NewHandler(http.HandlerFunc(handleHello), "GET /hello"))
	mux.Handle("GET /health", otelhttp.NewHandler(http.HandlerFunc(handleHealth), "GET /health"))
	return mux
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	slog.InfoContext(r.Context(), "served greeting", "name", name)
	writeJSON(r.Context(), w, http.StatusOK, helloResponse{Message: "Hello, " + name + "!"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(r.Context(), w, http.StatusOK, healthResponse{Status: "ok"})
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.ErrorContext(ctx, "encode response", "err", err)
	}
}
