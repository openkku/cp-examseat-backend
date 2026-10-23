package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/exam/presenter"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// Explorer reads room rosters and the explorer's filter options.
type Explorer interface {
	RoomRoster(ctx context.Context, q exam.ExploreQuery) ([]domain.Seat, error)
	Options(ctx context.Context, q exam.OptionsQuery) ([]string, error)
}

// Explore serves the room explorer: rosters and filter options.
type Explore struct {
	exams Explorer
	cache *httpx.ResponseCache
}

// NewExplore creates an Explore handler caching rosters in cache.
func NewExplore(exams Explorer, cache *httpx.ResponseCache) *Explore {
	return &Explore{exams: exams, cache: cache}
}

// Index handles GET /api/explore?round=&room=&date=&time=[&seat=].
func (h *Explore) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	round, room, date, timeParam, seat := q.Get("round"), q.Get("room"), q.Get("date"), q.Get("time"), q.Get("seat")

	if round == "" || room == "" || timeParam == "" || date == "" {
		httpx.Error(w, "Round, Room, Date, and Time parameters are required", http.StatusBadRequest)
		return
	}
	if httpx.ParamsTooLong(q) {
		httpx.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("explore:%s:%s:%s:%s:%s", round, room, date, timeParam, seat)
	err := h.cache.ServeJSON(w, r, key, func() (int, any, error) {
		query := exam.ExploreQuery{Round: round, Room: room, Date: date, Time: timeParam}
		if seat != "" {
			query.Seat = &seat
		}

		seats, err := h.exams.RoomRoster(r.Context(), query)
		if err != nil {
			return 0, nil, err
		}
		if len(seats) == 0 {
			return http.StatusNotFound, httpx.ErrorResponse{Error: "no exams found"}, nil
		}
		return http.StatusOK, presenter.ExamSchedules(seats), nil
	})

	if errors.Is(err, httpx.ErrRender) {
		log.Printf("Explore render error: %v", err)
		httpx.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else if err != nil {
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
	}
}

// Options handles GET /api/options?type={dates|times|rooms}&round=[&date=&time=].
func (h *Explore) Options(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	round := q.Get("round")

	if round == "" {
		httpx.Error(w, "Round parameter is required", http.StatusBadRequest)
		return
	}
	if httpx.ParamsTooLong(q) {
		httpx.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	options, err := h.exams.Options(r.Context(), exam.OptionsQuery{
		Mode:  q.Get("type"),
		Round: round,
		Date:  q.Get("date"),
		Time:  q.Get("time"),
	})
	if errors.Is(err, exam.ErrInvalidOptionMode) {
		httpx.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Options query error: %v", err)
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if len(options) == 0 {
		httpx.NotFound(w, "No options found")
		return
	}

	httpx.JSON(w, http.StatusOK, options)
}
