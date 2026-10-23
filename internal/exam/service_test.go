package exam

import (
	"context"
	"errors"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeRepo struct {
	Repository
	seats       []domain.Seat
	err         error
	lastSeatQ   domain.SeatQuery
	lastOptions OptionsQuery
}

func (f *fakeRepo) GetSeatsByID(ctx context.Context, q domain.SeatQuery) ([]domain.Seat, error) {
	f.lastSeatQ = q
	return f.seats, f.err
}

func (f *fakeRepo) GetOptions(ctx context.Context, q OptionsQuery) ([]string, error) {
	f.lastOptions = q
	return nil, f.err
}

type fakeRooms map[string]bool

func (r fakeRooms) Has(name string) bool { return r[name] }
func (r fakeRooms) Count() int           { return len(r) }

type fakeUpstream struct {
	NoUpstream
	err error
}

func (f fakeUpstream) Ensure(context.Context, string, string, string) error { return f.err }
func (f fakeUpstream) EnsureAll(context.Context, string, string) error      { return f.err }

func TestStudentScheduleNormalizesStudentID(t *testing.T) {
	repo := &fakeRepo{}
	round := "mid_1_2569"
	if _, err := NewService(repo, nil, nil).StudentSchedule(context.Background(), "653380123-4", round, ""); err != nil {
		t.Fatal(err)
	}
	if repo.lastSeatQ.StudentID != "6533801234" || *repo.lastSeatQ.Round != round {
		t.Errorf("query = %+v", repo.lastSeatQ)
	}
}

func TestStudentScheduleUpstreamErrors(t *testing.T) {
	ctx := context.Background()
	for _, err := range []error{&checkex.LookupQueuedError{Position: 1}, checkex.ErrLookupLimited, checkex.ErrLookupBusy, checkex.ErrUpstreamUnavailable} {
		if _, got := NewService(&fakeRepo{}, fakeUpstream{err: err}, nil).StudentSchedule(ctx, "1", "r", ""); !errors.Is(got, err) {
			t.Errorf("%v: got %v", err, got)
		}
	}
	// Other upstream failures fall through to the stored seats.
	repo := &fakeRepo{seats: []domain.Seat{{StudentID: "1"}}}
	if seats, err := NewService(repo, fakeUpstream{err: errors.New("vpn down")}, nil).StudentSchedule(ctx, "1", "r", ""); err != nil || len(seats) != 1 {
		t.Errorf("seats = %v, err = %v", seats, err)
	}
}

func TestCalendarSeatsIncompleteOnUpstreamError(t *testing.T) {
	ctx := context.Background()
	if _, complete, _ := NewService(&fakeRepo{}, nil, nil).CalendarSeats(ctx, "1", ""); !complete {
		t.Error("feed without upstream should be complete")
	}
	if _, complete, _ := NewService(&fakeRepo{}, fakeUpstream{err: errors.New("down")}, nil).CalendarSeats(ctx, "1", ""); complete {
		t.Error("feed built while upstream failed must not be complete")
	}
}

func TestOptionsRoomFilter(t *testing.T) {
	repo := &fakeRepo{}
	ctx := context.Background()

	// No rooms configured: every room is offered.
	NewService(repo, nil, fakeRooms{}).Options(ctx, OptionsQuery{Mode: OptionRooms})
	if repo.lastOptions.RoomFilter != nil {
		t.Error("RoomFilter should be unset when no room layouts are loaded")
	}

	NewService(repo, nil, fakeRooms{"CP.9127": true}).Options(ctx, OptionsQuery{Mode: OptionRooms})
	if f := repo.lastOptions.RoomFilter; f == nil || !f("CP.9127") || f("SC.1101") {
		t.Error("RoomFilter should accept only rooms with a loaded layout")
	}
}
