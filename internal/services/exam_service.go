package services

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// ExamService answers seat lookups: a student's schedule, a room's roster and
// the explorer's cascading filter options.
type ExamService struct {
	repo  models.ExamRepository
	rooms *RoomService
}

// NewExamService creates an ExamService. rooms may be nil, which disables
// hiding rooms without a layout from the explorer options.
func NewExamService(repo models.ExamRepository, rooms *RoomService) *ExamService {
	return &ExamService{repo: repo, rooms: rooms}
}

// StudentSchedule returns a student's seats, optionally limited to one round.
// studentID is normalized, so "653380123-4" and "6533801234" are equivalent.
func (s *ExamService) StudentSchedule(ctx context.Context, studentID string, round *string) ([]models.Seat, error) {
	return s.repo.GetSeatsByID(ctx, models.SeatQuery{
		StudentID: models.NormalizeStudentID(studentID),
		Round:     round,
	})
}

// RoomRoster returns the seats of one room and exam slot.
func (s *ExamService) RoomRoster(ctx context.Context, q models.ExploreQuery) ([]models.Seat, error) {
	return s.repo.GetSeats(ctx, q)
}

// Options lists the distinct dates, times or rooms of a round. When room
// layouts are configured, only rooms with a layout are considered so the
// explorer never offers a room it cannot draw.
func (s *ExamService) Options(ctx context.Context, q models.OptionsQuery) ([]string, error) {
	if s.rooms != nil && s.rooms.Count() > 0 {
		q.RoomFilter = s.rooms.Has
	}
	return s.repo.GetOptions(ctx, q)
}
