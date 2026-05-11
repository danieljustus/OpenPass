//go:build metrics

package serverbootstrap

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/danieljustus/OpenPass/internal/audit"
	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/metrics"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

// registerMetricsEndpoint registers the /metrics endpoint on the given mux.
// When built with the metrics tag, it serves Prometheus metrics with
// configurable bearer auth for non-loopback binds.
func registerMetricsEndpoint(mux *http.ServeMux, v *vaultpkg.Vault, bind string, legacyToken string, tokenRegistry *mcp.TokenRegistry, authAuditLog *audit.Logger) {
	h := promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})
	metricsAuthRequired := true
	if v != nil && v.Config != nil && v.Config.MCP != nil {
		metricsAuthRequired = v.Config.MCP.MetricsAuthRequired
	}
	if !mcp.IsLoopbackBind(bind) && metricsAuthRequired {
		mux.Handle("/metrics", mcp.BearerAuthMiddleware(legacyToken, tokenRegistry, authAuditLog, h))
	} else {
		mux.Handle("/metrics", h)
	}
}
