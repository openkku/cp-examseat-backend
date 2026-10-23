// Package handler serves the round list over HTTP.
package handler

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// Rounds lists the cached rounds.
type Rounds interface {
	List() []domain.Round
}

// Handler lists exam rounds.
type Handler struct {
	rounds Rounds
}

// New creates a Handler.
func New(rounds Rounds) *Handler {
	return &Handler{rounds: rounds}
}

// Index handles GET /api/rounds, returning [{id, label}] newest first.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.rounds.List())
}
