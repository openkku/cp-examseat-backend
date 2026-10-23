package checkex

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// flightTimeout bounds one shared upstream call.
const flightTimeout = 30 * time.Second

// Options configures on-demand lookups from checkex.
type Options struct {
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

// Service makes sure a student's seats for a checkex-backed round are
// in the database, fetching them through scrapler at most once. A nil
// *Service is valid and does nothing.
type Service struct {
	opts    Options
	repo    Repository
	source  Source
	limiter *notFoundLimiter
	flights singleflight.Group
}

// New returns nil when no rounds are configured, which disables the feature.
func New(opts Options, repo Repository, source Source) *Service {
	if len(opts.Rounds) == 0 {
		return nil
	}
	return &Service{
		opts:    opts,
		repo:    repo,
		source:  source,
		limiter: newNotFoundLimiter(opts.NotFoundLimit, opts.NotFoundWindow),
	}
}

// Enabled reports whether round is served by checkex.
func (s *Service) Enabled(round string) bool {
	return s != nil && slices.Contains(s.opts.Rounds, round)
}

// ValidID reports whether id is exactly a 10-digit student ID from an allowed
// intake year. Only valid IDs are ever sent upstream, which keeps us from
// probing arbitrary numbers.
func (s *Service) ValidID(id string) bool {
	if len(id) != 10 || domain.NormalizeStudentID(id) != id {
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
func (s *Service) RegisterRounds(ctx context.Context) error {
	if s == nil {
		return nil
	}
	for _, round := range s.opts.Rounds {
		label := s.opts.RoundLabels[round]
		if label == "" {
			label = domain.DefaultRoundLabel(round)
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
// in time, Ensure returns a *LookupQueuedError; scrapler finishes the
// lookup in the background and keeps the answer, so the visitor's next request
// stores it without a new upstream call.
func (s *Service) Ensure(ctx context.Context, round, studentID, client string) error {
	if !s.Enabled(round) || !s.ValidID(studentID) {
		return nil
	}
	if need, err := s.needsUpstream(ctx, round, studentID); err != nil || !need {
		return err
	}
	if s.limiter.blocked(client) {
		log.Printf("🚫 checkex: %s/%s refused, client %s hit the not-found limit", round, studentID, client)
		return ErrLookupLimited
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
func (s *Service) EnsureAll(ctx context.Context, studentID, client string) error {
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
func (s *Service) needsUpstream(ctx context.Context, round, studentID string) (bool, error) {
	l, err := s.repo.GetLookup(ctx, round, studentID)
	if err != nil {
		return false, err
	}
	if l != nil && (l.Status == LookupFound || time.Since(l.FetchedAt) < s.opts.NotFoundTTL) {
		return false, nil
	}
	if l == nil {
		// Seats imported some other way (Excel, JSON) are never asked upstream.
		// Custom datasets (out-of-schedule exams) don't count: upstream still
		// has the student's regular exams.
		imported, err := s.repo.HasRegularSeats(ctx, round, studentID)
		if err != nil || imported {
			return false, err
		}
	}
	return true, nil
}

// fetch asks scrapler and stores the answer.
func (s *Service) fetch(ctx context.Context, round, studentID, client string) error {
	seats, found, err := s.source.Lookup(ctx, round, studentID, client)
	var queued *LookupQueuedError
	switch {
	case errors.As(err, &queued), errors.Is(err, ErrLookupLimited), errors.Is(err, ErrLookupBusy):
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
