package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// flightTimeout bounds one shared upstream call.
const flightTimeout = 30 * time.Second

// CheckexOptions configures on-demand lookups from checkex.
type CheckexOptions struct {
	// Rounds served by checkex; empty disables the feature.
	Rounds []string
	// RoundLabels overrides the derived display label per round ID.
	RoundLabels map[string]string
	// IDPrefixes are the intake years whose 10-digit IDs may be sent upstream.
	IDPrefixes []string
	// NotFoundTTL is how long a "not found" answer is trusted.
	NotFoundTTL time.Duration
	// NotFoundLimit is the not-found answers one visitor may cause per
	// NotFoundWindow; 0 disables the limit.
	NotFoundLimit  int
	NotFoundWindow time.Duration
}

// Lookuper fetches one student's seats from upstream (the scrapler client).
type Lookuper interface {
	Lookup(ctx context.Context, roundID, studentID, client string) ([]models.Seat, bool, error)
}

// CheckexService makes sure a student's seats for a checkex-backed round are
// in the database, fetching them through scrapler at most once. A nil
// *CheckexService is valid and does nothing.
type CheckexService struct {
	opts    CheckexOptions
	repo    models.LookupRepository
	source  Lookuper
	limiter *notFoundLimiter
	flights singleflight.Group
}

// NewCheckexService returns nil when no rounds are configured, which disables the feature.
func NewCheckexService(opts CheckexOptions, repo models.LookupRepository, source Lookuper) *CheckexService {
	if len(opts.Rounds) == 0 {
		return nil
	}
	return &CheckexService{
		opts:    opts,
		repo:    repo,
		source:  source,
		limiter: newNotFoundLimiter(opts.NotFoundLimit, opts.NotFoundWindow),
	}
}

// Enabled reports whether round is served by checkex.
func (s *CheckexService) Enabled(round string) bool {
	return s != nil && slices.Contains(s.opts.Rounds, round)
}

// ValidID reports whether id is exactly a 10-digit student ID from an allowed
// intake year. Only valid IDs are ever sent upstream, which keeps us from
// probing arbitrary numbers.
func (s *CheckexService) ValidID(id string) bool {
	if len(id) != 10 || models.NormalizeStudentID(id) != id {
		return false
	}
	for _, p := range s.opts.IDPrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

// RegisterRounds makes configured rounds appear in the round list before any
// seats are fetched. Call it before loading rounds.
func (s *CheckexService) RegisterRounds(ctx context.Context) error {
	if s == nil {
		return nil
	}
	for _, round := range s.opts.Rounds {
		label := s.opts.RoundLabels[round]
		if label == "" {
			label = models.DefaultRoundLabel(round)
		}
		if err := s.repo.EnsureRound(ctx, round, label); err != nil {
			return err
		}
	}
	return nil
}

// Ensure fetches and stores the student's seats for round unless they are
// already known. client identifies the visitor (their IP) for the per-client
// limits; "" is never limited.
//
// scrapler paces upstream lookups through its queue. If the turn doesn't come
// in time, Ensure returns a *models.LookupQueuedError; scrapler finishes the
// lookup in the background and keeps the answer, so the visitor's next request
// stores it without a new upstream call.
func (s *CheckexService) Ensure(ctx context.Context, round, studentID, client string) error {
	if !s.Enabled(round) || !s.ValidID(studentID) {
		return nil
	}
	if need, err := s.needsUpstream(ctx, round, studentID); err != nil || !need {
		return err
	}
	if s.limiter.blocked(client) {
		log.Printf("🚫 checkex: %s/%s refused, client %s hit the not-found limit", round, studentID, client)
		return models.ErrLookupLimited
	}

	// Identical concurrent requests share one call. The call outlives the
	// request that started it, so a visitor leaving early doesn't fail the others.
	ch := s.flights.DoChan(round+"|"+studentID, func() (any, error) {
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), flightTimeout)
		defer cancel()
		// Another request may have stored this student while we waited.
		if need, err := s.needsUpstream(fctx, round, studentID); err != nil || !need {
			return nil, err
		}
		return nil, s.fetch(fctx, round, studentID, client)
	})
	select {
	case res := <-ch:
		return res.Err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// EnsureAll runs Ensure for every configured round (used by the round-less calendar feed).
func (s *CheckexService) EnsureAll(ctx context.Context, studentID, client string) error {
	if s == nil {
		return nil
	}
	var errs []error
	for _, round := range s.opts.Rounds {
		errs = append(errs, s.Ensure(ctx, round, studentID, client))
	}
	return errors.Join(errs...)
}

// needsUpstream reports whether the student's seats for round still have to be fetched.
func (s *CheckexService) needsUpstream(ctx context.Context, round, studentID string) (bool, error) {
	l, err := s.repo.GetLookup(ctx, round, studentID)
	if err != nil {
		return false, err
	}
	if l != nil && (l.Status == models.LookupFound || time.Since(l.FetchedAt) < s.opts.NotFoundTTL) {
		return false, nil
	}
	if l == nil {
		// Seats imported some other way (Excel, JSON) are never asked upstream.
		// Custom datasets (out-of-schedule exams) don't count: upstream still
		// has the student's regular exams.
		existing, err := s.repo.GetSeatsByID(ctx, models.SeatQuery{StudentID: studentID, Round: &round})
		if err != nil {
			return false, err
		}
		for _, seat := range existing {
			if seat.CustomID == "" {
				return false, nil
			}
		}
	}
	return true, nil
}

// fetch asks scrapler and stores the answer.
func (s *CheckexService) fetch(ctx context.Context, round, studentID, client string) error {
	seats, found, err := s.source.Lookup(ctx, round, studentID, client)
	var queued *models.LookupQueuedError
	switch {
	case errors.As(err, &queued), errors.Is(err, models.ErrLookupLimited), errors.Is(err, models.ErrLookupBusy):
		return err
	case err != nil:
		log.Printf("⚠️ checkex: lookup %s/%s failed: %v", round, studentID, err)
		return fmt.Errorf("checkex lookup: %w", err)
	case !found:
		log.Printf("🔎 checkex: %s/%s not found upstream", round, studentID)
		s.limiter.record(client)
		return s.repo.MarkNotFound(ctx, round, studentID)
	}
	log.Printf("🔎 checkex: %s/%s fetched %d seats", round, studentID, len(seats))
	return s.repo.AddStudentSeats(ctx, round, studentID, seats)
}
