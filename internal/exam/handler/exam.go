// Package handler serves the exam slice over HTTP: a student's schedule,
// the room explorer and per-student calendar feeds.
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/exam/presenter"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// queuedRetryAfter is how many seconds a queued visitor should wait before asking again.
const queuedRetryAfter = 5

// Schedules returns a student's seats in one round.
type Schedules interface {
	StudentSchedule(ctx context.Context, studentID, round, client string) ([]domain.Seat, error)
}

// Exam serves a student's exam schedule.
type Exam struct {
	exams    Schedules
	clientIP httpx.ClientIPFunc
}

// NewExam creates an Exam handler.
func NewExam(exams Schedules, clientIP httpx.ClientIPFunc) *Exam {
	return &Exam{exams: exams, clientIP: clientIP}
}

// Show handles GET /api/exam?id={studentID}&round={roundID}.
func (h *Exam) Show(w http.ResponseWriter, r *http.Request) {
	id := domain.NormalizeStudentID(r.URL.Query().Get("id"))
	round := r.URL.Query().Get("round")

	if id == "" || round == "" {
		httpx.Error(w, "Student ID and Round are required", http.StatusBadRequest)
		return
	}
	if len(id) > httpx.MaxStudentIDDigits || httpx.ParamsTooLong(r.URL.Query()) {
		httpx.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	// For rounds served by checkex, the student's seats are fetched on first request.
	seats, err := h.exams.StudentSchedule(r.Context(), id, round, httpx.ClientIP(h.clientIP, r))
	var queued *checkex.LookupQueuedError
	switch {
	case errors.As(err, &queued):
		// Still in line; the lookup finishes in the background and the retry reads the DB.
		writeQueued(w, queued.Position, queuedRetryAfter)
		return
	case errors.Is(err, checkex.ErrLookupLimited):
		httpx.Error(w, "ค้นหารหัสที่ไม่พบข้อมูลบ่อยเกินไป กรุณาลองใหม่ภายหลัง", http.StatusTooManyRequests)
		return
	case errors.Is(err, checkex.ErrLookupBusy):
		httpx.Error(w, "ระบบกำลังดึงข้อมูล กรุณาลองใหม่อีกครั้ง", http.StatusServiceUnavailable)
		return
	case err != nil:
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if len(seats) == 0 {
		httpx.NotFound(w, "No exam schedules found")
		return
	}

	httpx.JSON(w, http.StatusOK, presenter.ExamSchedules(seats))
}

// QueuedResponse is the 202 body of a lookup still waiting in the checkex queue.
type QueuedResponse struct {
	Queued     bool `json:"queued"`
	Position   int  `json:"position"`
	RetryAfter int  `json:"retry_after"`
}

// writeQueued tells the client its lookup is still queued upstream and when to ask again.
func writeQueued(w http.ResponseWriter, position int, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	httpx.JSON(w, http.StatusAccepted, QueuedResponse{Queued: true, Position: position, RetryAfter: retryAfter})
}
