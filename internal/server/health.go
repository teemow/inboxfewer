package server

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Health status constants for health check responses.
const (
	healthStatusOK           = "ok"
	healthStatusNotReady     = "not ready"
	healthStatusShuttingDown = "shutting down"
)

// HealthChecker provides health check endpoints for Kubernetes probes.
type HealthChecker struct {
	// ready indicates whether the server is ready to receive traffic
	ready atomic.Bool
	// serverContext provides access to dependencies for health checks
	serverContext *ServerContext
	// startTime tracks when the server started
	startTime time.Time
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(sc *ServerContext) *HealthChecker {
	h := &HealthChecker{
		serverContext: sc,
		startTime:     time.Now(),
	}
	// Server starts as ready by default
	h.ready.Store(true)
	return h
}

// SetReady sets the readiness state of the server.
func (h *HealthChecker) SetReady(ready bool) {
	h.ready.Store(ready)
}

// IsReady returns whether the server is ready to receive traffic.
func (h *HealthChecker) IsReady() bool {
	return h.ready.Load()
}

// isServerShuttingDown checks if the server context is shutting down.
// Returns false if serverContext is nil (safe for testing).
func (h *HealthChecker) isServerShuttingDown() bool {
	return h.serverContext != nil && h.serverContext.IsShutdown()
}

// HealthResponse represents the JSON response for health endpoints.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// DetailedHealthResponse provides comprehensive health information.
type DetailedHealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// LivenessHandler returns an HTTP handler for the /healthz endpoint.
// Liveness probes indicate whether the process should be restarted.
// This should be a simple check that the server process is running.
func (h *HealthChecker) LivenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := HealthResponse{
			Status: healthStatusOK,
		}

		_ = json.NewEncoder(w).Encode(response)
	})
}

// ReadinessHandler returns an HTTP handler for the /readyz endpoint.
// Readiness probes indicate whether the server is ready to receive traffic.
func (h *HealthChecker) ReadinessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		checks := make(map[string]string)
		allOk := true

		// Check if server is marked as ready
		if !h.ready.Load() {
			checks["ready"] = healthStatusNotReady
			allOk = false
		} else {
			checks["ready"] = healthStatusOK
		}

		// Check if server context is not shutdown
		if h.isServerShuttingDown() {
			checks["shutdown"] = healthStatusShuttingDown
			allOk = false
		} else {
			checks["shutdown"] = healthStatusOK
		}

		response := HealthResponse{
			Checks: checks,
		}

		if allOk {
			response.Status = healthStatusOK
			w.WriteHeader(http.StatusOK)
		} else {
			response.Status = healthStatusNotReady
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	})
}

// RegisterHealthEndpoints registers health check endpoints on the given mux.
func (h *HealthChecker) RegisterHealthEndpoints(mux *http.ServeMux) {
	mux.Handle("/healthz", h.LivenessHandler())
	mux.Handle("/readyz", h.ReadinessHandler())
	mux.Handle("/healthz/detailed", h.DetailedHealthHandler())
}

// DetailedHealthHandler returns an HTTP handler for the /healthz/detailed endpoint.
// This endpoint provides comprehensive health information.
func (h *HealthChecker) DetailedHealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := DetailedHealthResponse{
			Status: healthStatusOK,
			Uptime: time.Since(h.startTime).Truncate(time.Second).String(),
		}

		// Determine overall status
		if !h.ready.Load() {
			response.Status = healthStatusNotReady
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if h.isServerShuttingDown() {
			response.Status = healthStatusShuttingDown
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		_ = json.NewEncoder(w).Encode(response)
	})
}
