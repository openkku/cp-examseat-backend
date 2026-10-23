package services

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
)

const (
	fakeID    = "6833800999"
	testRound = "final_1_2569"
)

// fakeSeats is what scrapler answers for fakeID.
var fakeSeats = []models.Seat{
	{Sheet: "19 ต.ค. 69 (เช้า)", Date: "2026-10-19", TimeStart: "08.30", TimeEnd: "11.30", Room: "SC.7401", Subject: "LI102003", SubjectName: "ENGLISH III", Section: "31", Seat: "B11", Branch: "CP-Cy"},
	{Sheet: "20 ต.ค. 69 (เช้า)", Date: "2026-10-20", TimeStart: "08.30", TimeEnd: "11.30", Room: "SC.7401", Subject: "CP422022", SubjectName: "Database Architecture Analysis and Design", Section: "02", Seat: "B51", Branch: "CP-Cy"},
	{Sheet: "22 ต.ค. 69 (บ่าย)", Date: "2026-10-22", TimeStart: "13.00", TimeEnd: "16.00", Room: "CP.9127", Subject: "CP422021", SubjectName: "Web and Mobile Application Architecture", Section: "01", Seat: "B9", Branch: "CP-Cy"},
}

type lookupFunc func(ctx context.Context, roundID, studentID, client string) ([]models.Seat, bool, error)

func (f lookupFunc) Lookup(ctx context.Context, roundID, studentID, client string) ([]models.Seat, bool, error) {
	return f(ctx, roundID, studentID, client)
}

func found(ctx context.Context, round, studentID, client string) ([]models.Seat, bool, error) {
	seats := make([]models.Seat, len(fakeSeats))
	for i, s := range fakeSeats {
		s.Category, s.StudentID, s.ExamRound = models.CategoryInSchedule, studentID, round
		seats[i] = s
	}
	return seats, true, nil
}

func id(n int) string { return fmt.Sprintf("68338%05d", n) }

func testCheckex(t *testing.T, opts CheckexOptions, lookup lookupFunc) (*CheckexService, *sqlite.Database) {
	t.Helper()
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	opts.Rounds = []string{testRound}
	opts.IDPrefixes = []string{"68", "69"}
	opts.NotFoundTTL = time.Hour
	s := NewCheckexService(opts, db, lookup)
	if err := s.RegisterRounds(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s, db
}

func storedSeats(t *testing.T, db *sqlite.Database, studentID string) []models.Seat {
	t.Helper()
	round := testRound
	seats, err := db.GetSeatsByID(context.Background(), models.SeatQuery{StudentID: studentID, Round: &round})
	if err != nil {
		t.Fatal(err)
	}
	return seats
}

func TestCheckexValidID(t *testing.T) {
	s := NewCheckexService(CheckexOptions{Rounds: []string{testRound}, IDPrefixes: []string{"68", "69"}}, nil, nil)
	cases := map[string]bool{
		"6833809314":  true,
		"6912345678":  true,
		"683380931":   false, // 9 digits
		"68338093141": false, // 11 digits
		"6712345678":  false, // old intake
		"7012345678":  false, // 2570 intake: needs prefix bump
		"68338a9314":  false,
		"683380931-4": false, // callers must normalise first
		"":            false,
	}
	for id, want := range cases {
		if got := s.ValidID(id); got != want {
			t.Errorf("ValidID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestCheckexDisabled(t *testing.T) {
	var s *CheckexService = NewCheckexService(CheckexOptions{}, nil, nil)
	if s != nil {
		t.Fatal("expected nil service without rounds")
	}
	if err := s.Ensure(context.Background(), testRound, fakeID, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureAll(context.Background(), fakeID, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterRounds(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCheckexFetchesOnce(t *testing.T) {
	var calls atomic.Int32
	r, db := testCheckex(t, CheckexOptions{}, func(ctx context.Context, round, studentID, client string) ([]models.Seat, bool, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		if studentID != fakeID {
			return nil, false, nil
		}
		return found(ctx, round, studentID, client)
	})
	ctx := context.Background()

	// Concurrent identical requests collapse into one call.
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); r.Ensure(ctx, testRound, fakeID, "") }()
	}
	wg.Wait()
	// Later requests are served from the DB.
	r.Ensure(ctx, testRound, fakeID, "")
	if calls.Load() != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls.Load())
	}
	if seats := storedSeats(t, db, fakeID); len(seats) != 3 {
		t.Fatalf("expected 3 stored seats, got %d", len(seats))
	}

	// Not-found is cached within the TTL.
	r.Ensure(ctx, testRound, "6933800000", "")
	r.Ensure(ctx, testRound, "6933800000", "")
	if calls.Load() != 2 {
		t.Fatalf("expected not-found to be cached, calls=%d", calls.Load())
	}

	// Invalid IDs and unconfigured rounds never go upstream.
	r.Ensure(ctx, testRound, "5733800000", "")
	r.Ensure(ctx, "mid_1_2569", fakeID, "")
	if calls.Load() != 2 {
		t.Fatalf("unexpected upstream call, calls=%d", calls.Load())
	}
}

func TestCheckexDoesNotCacheErrorsOrQueued(t *testing.T) {
	ctx := context.Background()
	for _, upstreamErr := range []error{errors.New("vpn down"), &models.LookupQueuedError{Position: 2}, models.ErrLookupBusy, models.ErrLookupLimited} {
		r, db := testCheckex(t, CheckexOptions{}, func(context.Context, string, string, string) ([]models.Seat, bool, error) {
			return nil, false, upstreamErr
		})
		if err := r.Ensure(ctx, testRound, fakeID, ""); err == nil {
			t.Fatalf("%v: expected error", upstreamErr)
		}
		if l, _ := db.GetLookup(ctx, testRound, fakeID); l != nil {
			t.Fatalf("%v must not be cached, got %+v", upstreamErr, l)
		}
	}

	var queued *models.LookupQueuedError
	r, _ := testCheckex(t, CheckexOptions{}, func(context.Context, string, string, string) ([]models.Seat, bool, error) {
		return nil, false, &models.LookupQueuedError{Position: 2}
	})
	if err := r.Ensure(ctx, testRound, fakeID, ""); !errors.As(err, &queued) || queued.Position != 2 {
		t.Fatalf("expected QueuedError to reach the handler, got %v", err)
	}
}

func TestCheckexFetchesAlongsideCustomSeats(t *testing.T) {
	var calls atomic.Int32
	r, db := testCheckex(t, CheckexOptions{}, func(ctx context.Context, round, studentID, client string) ([]models.Seat, bool, error) {
		calls.Add(1)
		return found(ctx, round, studentID, client)
	})
	ctx := context.Background()
	custom := []models.Seat{{
		Sheet: "นัดสอบปลายภาค (บรรยาย)", Date: "2026-10-17", TimeStart: "13.00", TimeEnd: "16.00",
		CustomID: "FINAL_1_2569_CP421025_LEC", Room: "CP9226", Subject: "CP421025", Section: "1",
		StudentID: fakeID, Seat: "CP9226-1", Branch: "CP-Cy", ExamRound: testRound,
	}}
	if err := db.AddRound(ctx, testRound, "ปลายภาค 1/2569", custom); err != nil {
		t.Fatal(err)
	}

	if err := r.Ensure(ctx, testRound, fakeID, ""); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected custom seats not to block upstream, calls=%d", calls.Load())
	}
	if seats := storedSeats(t, db, fakeID); len(seats) != 4 {
		t.Fatalf("expected 1 custom + 3 upstream seats, got %d", len(seats))
	}
}

func TestNotFoundLimitPerClient(t *testing.T) {
	var calls atomic.Int32
	r, db := testCheckex(t, CheckexOptions{NotFoundLimit: 2, NotFoundWindow: time.Hour}, func(context.Context, string, string, string) ([]models.Seat, bool, error) {
		calls.Add(1)
		return nil, false, nil
	})
	ctx := context.Background()

	for i := range 2 {
		if err := r.Ensure(ctx, testRound, id(i), "a"); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Ensure(ctx, testRound, id(2), "a"); !errors.Is(err, models.ErrLookupLimited) {
		t.Fatalf("expected ErrLimited, got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("limited lookup went upstream, calls=%d", calls.Load())
	}
	if l, _ := db.GetLookup(ctx, testRound, id(2)); l != nil {
		t.Fatalf("limited lookup must not be cached, got %+v", l)
	}

	// Already-known IDs are still answered for a limited client (from the DB, no error).
	if err := r.Ensure(ctx, testRound, id(0), "a"); err != nil {
		t.Fatalf("cached lookup for limited client: %v", err)
	}
	// Other clients and unknown clients are unaffected.
	if err := r.Ensure(ctx, testRound, id(3), "b"); err != nil {
		t.Fatalf("client b: %v", err)
	}
	for i := 4; i < 8; i++ {
		if err := r.Ensure(ctx, testRound, id(i), ""); err != nil {
			t.Fatalf("empty client: %v", err)
		}
	}
}

func TestNotFoundLimitIgnoresFound(t *testing.T) {
	r, _ := testCheckex(t, CheckexOptions{NotFoundLimit: 1, NotFoundWindow: time.Hour}, found)
	for i := range 3 {
		if err := r.Ensure(context.Background(), testRound, id(i), "a"); err != nil {
			t.Fatalf("found lookup %d: %v", i, err)
		}
	}
}

func TestEnsureReturnsWhenCallerLeaves(t *testing.T) {
	release := make(chan struct{})
	r, db := testCheckex(t, CheckexOptions{}, func(ctx context.Context, round, studentID, client string) ([]models.Seat, bool, error) {
		<-release
		return found(ctx, round, studentID, client)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := r.Ensure(ctx, testRound, fakeID, ""); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the caller's deadline, got %v", err)
	}
	// The shared call keeps going and still stores the seats.
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for len(storedSeats(t, db, fakeID)) != 3 {
		if time.Now().After(deadline) {
			t.Fatal("detached lookup never stored seats")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
