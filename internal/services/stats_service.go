package services

import (
	"context"
	"log"
	"sync"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// StatsService precomputes the analytics dashboard once at startup; the data
// only changes when a new round is imported (followed by a restart).
type StatsService struct {
	repo      models.ExamRepository
	mu        sync.RWMutex
	dashboard *models.Dashboard
}

// NewStatsService creates a StatsService. Call Refresh before use.
func NewStatsService(repo models.ExamRepository) *StatsService {
	return &StatsService{repo: repo}
}

// Refresh rebuilds the dashboard from every seat in the repository,
// labelling rounds with roundLabels.
func (s *StatsService) Refresh(ctx context.Context, roundLabels map[string]string) error {
	log.Println("📊 Loading data for analytics...")
	seats, err := s.repo.GetAllSeats(ctx)
	if err != nil {
		return err
	}

	dashboard := models.BuildDashboard(seats, roundLabels)

	s.mu.Lock()
	s.dashboard = dashboard
	s.mu.Unlock()

	log.Printf("✅ Stats ready for %d options", len(dashboard.Options))
	return nil
}

// Dashboard returns the precomputed dashboard, or nil if it is not ready.
func (s *StatsService) Dashboard() *models.Dashboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dashboard
}
