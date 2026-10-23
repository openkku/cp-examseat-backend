// Package handler serves the analytics dashboard over HTTP.
package handler

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
	"github.com/openkku/cp-examseat-backend/internal/stats"
)

// Dashboards returns the precomputed dashboard.
type Dashboards interface {
	Dashboard() *stats.Dashboard
}

// Handler serves the analytics dashboard.
type Handler struct {
	stats Dashboards
}

// New creates a Handler.
func New(stats Dashboards) *Handler {
	return &Handler{stats: stats}
}

// Index handles GET /api/stats.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	dashboard := h.stats.Dashboard()
	if dashboard == nil {
		httpx.Error(w, "Stats not ready", http.StatusServiceUnavailable)
		return
	}
	httpx.JSON(w, http.StatusOK, dashboard)
}
