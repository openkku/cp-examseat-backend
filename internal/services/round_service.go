package services

import (
	"context"
	"log"
	"sync"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// RoundService keeps the list of exam rounds in memory so /api/rounds never
// hits the database.
type RoundService struct {
	repo   models.ExamRepository
	mu     sync.RWMutex
	rounds []models.Round
}

// NewRoundService creates a RoundService backed by repo. Call Load before use.
func NewRoundService(repo models.ExamRepository) *RoundService {
	return &RoundService{repo: repo, rounds: []models.Round{}}
}

// Load (re)reads every round from the repository.
func (s *RoundService) Load(ctx context.Context) error {
	rounds, err := s.repo.GetRounds(ctx)
	if err != nil {
		return err
	}
	if rounds == nil {
		rounds = []models.Round{}
	}

	s.mu.Lock()
	s.rounds = rounds
	s.mu.Unlock()

	log.Printf("✅ Loaded %d exam rounds into memory", len(rounds))
	return nil
}

// List returns the cached rounds, newest first.
func (s *RoundService) List() []models.Round {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rounds
}

// Labels maps round IDs to their display labels.
func (s *RoundService) Labels() map[string]string {
	labels := make(map[string]string)
	for _, r := range s.List() {
		labels[r.ID] = r.Label
	}
	return labels
}
