package stats

import (
	"context"
	"log"
	"sync"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// SeatSource reads every stored seat.
type SeatSource interface {
	GetAllSeats(ctx context.Context) ([]domain.Seat, error)
}

// Service precomputes the analytics dashboard; it is rebuilt on every
// (re)load, as the data only changes when rounds are imported or deleted.
type Service struct {
	seats     SeatSource
	mu        sync.RWMutex
	dashboard *Dashboard
}

// NewService creates a Service. Call Refresh before use.
func NewService(seats SeatSource) *Service {
	return &Service{seats: seats}
}

// Refresh rebuilds the dashboard from every seat in the repository,
// labelling rounds with roundLabels.
func (s *Service) Refresh(ctx context.Context, roundLabels map[string]string) error {
	log.Println("📊 Loading data for analytics...")
	seats, err := s.seats.GetAllSeats(ctx)
	if err != nil {
		return err
	}

	dashboard := BuildDashboard(seats, roundLabels)

	s.mu.Lock()
	s.dashboard = dashboard
	s.mu.Unlock()

	log.Printf("✅ Stats ready for %d options", len(dashboard.Options))
	return nil
}

// Dashboard returns the precomputed dashboard, or nil if it is not ready.
func (s *Service) Dashboard() *Dashboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dashboard
}
