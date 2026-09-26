package controllers

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// RoundController lists exam rounds.
type RoundController struct {
	rounds *services.RoundService
}

// NewRoundController creates a RoundController.
func NewRoundController(rounds *services.RoundService) *RoundController {
	return &RoundController{rounds: rounds}
}

// Index handles GET /api/rounds, returning [{id, label}] newest first.
func (c *RoundController) Index(w http.ResponseWriter, r *http.Request) {
	views.JSON(w, http.StatusOK, c.rounds.List())
}
