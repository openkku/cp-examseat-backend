package exam

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Service implements the exam use cases.
type Service struct {
	repo     Repository
	upstream Upstream
	rooms    RoomDirectory
}

// NewService creates a Service. upstream may be nil (no on-demand lookups) and
// rooms may be nil, which disables hiding rooms without a layout from the
// explorer options.
func NewService(repo Repository, upstream Upstream, rooms RoomDirectory) *Service {
	if upstream == nil {
		upstream = NoUpstream{}
	}
	return &Service{repo: repo, upstream: upstream, rooms: rooms}
}

// StudentSchedule returns a student's seats in one round. studentID is
// normalized, so "653380123-4" and "6533801234" are equivalent.
//
// For rounds fetched on demand the student's seats are fetched first. A
// lookup the visitor has to retry (*checkex.LookupQueuedError,
// checkex.ErrLookupLimited, checkex.ErrLookupBusy,
// checkex.ErrUpstreamUnavailable) is returned as the error;
// other upstream failures fall through to the stored seats.
func (s *Service) StudentSchedule(ctx context.Context, studentID, round, client string) ([]domain.Seat, error) {
	id := domain.NormalizeStudentID(studentID)
	if err := s.upstream.Ensure(ctx, round, id, client); RetryLater(err) {
		return nil, err
	}
	return s.repo.GetSeatsByID(ctx, domain.SeatQuery{StudentID: id, Round: &round})
}

// CalendarSeats returns a student's seats in every round. Feeds aren't
// round-scoped, so every on-demand round is fetched first; complete is false
// when one of those fetches failed, and the result should not be cached.
func (s *Service) CalendarSeats(ctx context.Context, studentID, client string) (seats []domain.Seat, complete bool, err error) {
	id := domain.NormalizeStudentID(studentID)
	upstreamErr := s.upstream.EnsureAll(ctx, id, client)
	seats, err = s.repo.GetSeatsByID(ctx, domain.SeatQuery{StudentID: id})
	return seats, upstreamErr == nil, err
}

// RoomRoster returns the seats of one room and exam slot.
func (s *Service) RoomRoster(ctx context.Context, q ExploreQuery) ([]domain.Seat, error) {
	return s.repo.GetSeats(ctx, q)
}

// Options lists the distinct dates, times or rooms of a round. When room
// layouts are configured, only rooms with a layout are considered so the
// explorer never offers a room it cannot draw.
func (s *Service) Options(ctx context.Context, q OptionsQuery) ([]string, error) {
	if s.rooms != nil && s.rooms.Count() > 0 {
		q.RoomFilter = s.rooms.Has
	}
	return s.repo.GetOptions(ctx, q)
}
