// Package metrics provides Prometheus metrics for OpenPass MCP server.
//
// It instruments MCP tool calls, authentication denials, approval outcomes,
// and vault operations with counters and histograms suitable for monitoring
// and alerting in production deployments.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var registry = prometheus.NewRegistry()

func init() {
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

var (
	mcpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openpass",
			Subsystem: "mcp",
			Name:      "requests_total",
			Help:      "Total number of MCP tool requests.",
		},
		[]string{"tool", "agent", "status"},
	)

	mcpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "openpass",
			Subsystem: "mcp",
			Name:      "request_duration_seconds",
			Help:      "Duration of MCP tool requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"tool", "agent"},
	)
)

var (
	mcpAuthDenialsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openpass",
			Subsystem: "mcp",
			Name:      "auth_denials_total",
			Help:      "Total number of MCP authentication/authorization denials.",
		},
		[]string{"reason", "agent"},
	)
)

var (
	mcpApprovalsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openpass",
			Subsystem: "mcp",
			Name:      "approvals_total",
			Help:      "Total number of MCP approval outcomes.",
		},
		[]string{"agent", "outcome"},
	)
)

var (
	vaultOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "openpass",
			Subsystem: "vault",
			Name:      "operations_total",
			Help:      "Total number of vault operations.",
		},
		[]string{"operation", "status"},
	)
)

func init() {
	registry.MustRegister(
		mcpRequestsTotal,
		mcpRequestDuration,
		mcpAuthDenialsTotal,
		mcpApprovalsTotal,
		vaultOperationsTotal,
	)
}

// RecordMCPRequest records an MCP tool request with its duration.
// status should be "success" or "error".
func RecordMCPRequest(tool, agent, status string, duration time.Duration) {
	mcpRequestsTotal.WithLabelValues(tool, agent, status).Inc()
	mcpRequestDuration.WithLabelValues(tool, agent).Observe(duration.Seconds())
}

// RecordAuthDenial records an authentication or authorization denial.
// reason should describe why access was denied (e.g., "scope_denied", "write_denied").
func RecordAuthDenial(reason, agent string) {
	mcpAuthDenialsTotal.WithLabelValues(reason, agent).Inc()
}

// RecordApproval records an approval outcome for a write operation.
// outcome should be "granted" or "denied".
func RecordApproval(agent, outcome string) {
	mcpApprovalsTotal.WithLabelValues(agent, outcome).Inc()
}

// RecordVaultOperation records a vault operation.
// operation describes the action (e.g., "read", "write", "delete").
// status should be "success" or "error".
func RecordVaultOperation(operation, status string) {
	vaultOperationsTotal.WithLabelValues(operation, status).Inc()
}

// Registry returns the Prometheus registry used by OpenPass.
// Use this with promhttp.HandlerFor to serve metrics over HTTP.
func Registry() *prometheus.Registry {
	return registry
}
