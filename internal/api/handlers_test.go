package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	h := NewHandler()

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantBody   string
	}{
		{"hello default", "/hello", http.StatusOK, `{"message":"Hello, World!"}`},
		{"hello named", "/hello?name=Ada", http.StatusOK, `{"message":"Hello, Ada!"}`},
		{"health", "/health", http.StatusOK, `{"status":"ok"}`},
		{"unknown route", "/nope", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
					t.Errorf("body = %q, want %q", got, tt.wantBody)
				}
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
		})
	}
}
