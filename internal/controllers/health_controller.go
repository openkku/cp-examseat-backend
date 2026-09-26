package controllers

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/views"
)

// Health handles GET /healthz for container and load-balancer probes.
func Health(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
