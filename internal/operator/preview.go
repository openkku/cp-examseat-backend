package operator

import (
	"context"
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// ExamPreviewer queries real student rosters and filter options for previews.
type ExamPreviewer interface {
	Options(ctx context.Context, q exam.OptionsQuery) ([]string, error)
	RoomRoster(ctx context.Context, q exam.ExploreQuery) ([]domain.Seat, error)
}

// PreviewSeat is the student seat projection returned for layout previewing.
type PreviewSeat struct {
	Seat      string `json:"seat"`
	StudentID string `json:"student_id"`
	Subject   string `json:"subject"`
	Section   string `json:"section"`
}

// PreviewHandler handles /api/preview/* routes.
type PreviewHandler struct {
	exams ExamPreviewer
}

// NewPreviewHandler creates a PreviewHandler.
func NewPreviewHandler(exams ExamPreviewer) *PreviewHandler {
	return &PreviewHandler{exams: exams}
}

// Options handles GET /api/preview/options?type=dates|times&round=&room=[&date=].
func (h *PreviewHandler) Options(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode := q.Get("type")
	round := q.Get("round")
	room := q.Get("room")
	date := q.Get("date")
	timeParam := q.Get("time")

	if round == "" || (mode != exam.OptionDates && mode != exam.OptionTimes) {
		httpx.Error(w, "Round and type (dates|times) are required", http.StatusBadRequest)
		return
	}

	query := exam.OptionsQuery{
		Mode:  mode,
		Round: round,
		Date:  date,
		Time:  timeParam,
	}
	if room != "" {
		query.RoomFilter = func(r string) bool { return r == room }
	}

	options, err := h.exams.Options(r.Context(), query)
	if err != nil {
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	if options == nil {
		options = []string{}
	}
	httpx.JSON(w, http.StatusOK, options)
}

// Seats handles GET /api/preview/seats?round=&room=&date=&time=.
func (h *PreviewHandler) Seats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	round := q.Get("round")
	room := q.Get("room")
	date := q.Get("date")
	timeParam := q.Get("time")

	if round == "" || room == "" || date == "" || timeParam == "" {
		httpx.Error(w, "Round, room, date, and time parameters are required", http.StatusBadRequest)
		return
	}

	seats, err := h.exams.RoomRoster(r.Context(), exam.ExploreQuery{
		Round: round,
		Room:  room,
		Date:  date,
		Time:  timeParam,
	})
	if err != nil {
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	res := make([]PreviewSeat, len(seats))
	for i, s := range seats {
		res[i] = PreviewSeat{
			Seat:      s.Seat,
			StudentID: s.StudentID,
			Subject:   s.Subject,
			Section:   s.Section,
		}
	}
	httpx.JSON(w, http.StatusOK, res)
}
