package httpx

import (
	"net/http"
)

// Health handles GET /healthz for container and load-balancer probes.
func Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
