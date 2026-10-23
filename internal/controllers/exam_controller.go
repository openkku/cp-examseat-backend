package controllers

import (
	"errors"
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// queuedRetryAfter is how many seconds a queued visitor should wait before asking again.
const queuedRetryAfter = 5

// ClientIPFunc identifies the visitor of a request for per-client limits.
type ClientIPFunc func(*http.Request) string

// ExamController serves a student's exam schedule.
type ExamController struct {
	exams    *services.ExamService
	checkex  *services.CheckexService
	clientIP ClientIPFunc
}

// NewExamController creates an ExamController. checkex may be nil, which
// disables on-demand lookups from checkex.
func NewExamController(exams *services.ExamService, checkex *services.CheckexService, clientIP ClientIPFunc) *ExamController {
	return &ExamController{exams: exams, checkex: checkex, clientIP: clientIP}
}

// Show handles GET /api/exam?id={studentID}&round={roundID}.
func (c *ExamController) Show(w http.ResponseWriter, r *http.Request) {
	id := models.NormalizeStudentID(r.URL.Query().Get("id"))
	round := r.URL.Query().Get("round")

	if id == "" || round == "" {
		views.Error(w, "Student ID and Round are required", http.StatusBadRequest)
		return
	}
	if len(id) > maxStudentIDDigits || paramsTooLong(r.URL.Query()) {
		views.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	// For rounds served by checkex, fetch this student's seats on first request.
	// Other upstream failures fall through to the DB read and show as "no data".
	var queued *models.LookupQueuedError
	switch err := c.checkex.Ensure(r.Context(), round, id, clientIP(c.clientIP, r)); {
	case errors.As(err, &queued):
		// Still in line; the lookup finishes in the background and the retry reads the DB.
		views.Queued(w, queued.Position, queuedRetryAfter)
		return
	case errors.Is(err, models.ErrLookupLimited):
		views.Error(w, "ค้นหารหัสที่ไม่พบข้อมูลบ่อยเกินไป กรุณาลองใหม่ภายหลัง", http.StatusTooManyRequests)
		return
	case errors.Is(err, models.ErrLookupBusy):
		views.Error(w, "ระบบกำลังดึงข้อมูล กรุณาลองใหม่อีกครั้ง", http.StatusServiceUnavailable)
		return
	}

	seats, err := c.exams.StudentSchedule(r.Context(), id, &round)
	if err != nil {
		views.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if len(seats) == 0 {
		views.NotFound(w, "No exam schedules found")
		return
	}

	views.JSON(w, http.StatusOK, views.ExamSchedules(seats))
}

// clientIP returns the visitor's address, or "" (never limited) without a resolver.
func clientIP(f ClientIPFunc, r *http.Request) string {
	if f == nil {
		return ""
	}
	return f(r)
}
