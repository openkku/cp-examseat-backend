package controllers

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// StatsController serves the analytics dashboard.
type StatsController struct {
	stats *services.StatsService
}

// NewStatsController creates a StatsController.
func NewStatsController(stats *services.StatsService) *StatsController {
	return &StatsController{stats: stats}
}

// Index handles GET /api/stats.
func (c *StatsController) Index(w http.ResponseWriter, r *http.Request) {
	dashboard := c.stats.Dashboard()
	if dashboard == nil {
		views.Error(w, "Stats not ready", http.StatusServiceUnavailable)
		return
	}
	views.JSON(w, http.StatusOK, dashboard)
}
