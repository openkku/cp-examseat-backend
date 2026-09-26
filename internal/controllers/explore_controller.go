package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// ExploreController serves the room explorer: rosters and filter options.
type ExploreController struct {
	exams *services.ExamService
	cache *views.ResponseCache
}

// NewExploreController creates an ExploreController caching rosters in cache.
func NewExploreController(exams *services.ExamService, cache *views.ResponseCache) *ExploreController {
	return &ExploreController{exams: exams, cache: cache}
}

// Index handles GET /api/explore?round=&room=&date=&time=[&seat=].
func (c *ExploreController) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	round, room, date, timeParam, seat := q.Get("round"), q.Get("room"), q.Get("date"), q.Get("time"), q.Get("seat")

	if round == "" || room == "" || timeParam == "" || date == "" {
		views.Error(w, "Round, Room, Date, and Time parameters are required", http.StatusBadRequest)
		return
	}
	if paramsTooLong(q) {
		views.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("explore:%s:%s:%s:%s:%s", round, room, date, timeParam, seat)
	err := c.cache.ServeJSON(w, r, key, func() (int, any, error) {
		query := models.ExploreQuery{Round: round, Room: room, Date: date, Time: timeParam}
		if seat != "" {
			query.Seat = &seat
		}

		seats, err := c.exams.RoomRoster(r.Context(), query)
		if err != nil {
			return 0, nil, err
		}
		if len(seats) == 0 {
			return http.StatusNotFound, views.ErrorResponse{Error: "no exams found"}, nil
		}
		return http.StatusOK, views.ExamSchedules(seats), nil
	})

	if errors.Is(err, views.ErrRender) {
		log.Printf("Explore render error: %v", err)
		views.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else if err != nil {
		views.Error(w, "Database Error", http.StatusInternalServerError)
	}
}

// Options handles GET /api/options?type={dates|times|rooms}&round=[&date=&time=].
func (c *ExploreController) Options(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	round := q.Get("round")

	if round == "" {
		views.Error(w, "Round parameter is required", http.StatusBadRequest)
		return
	}
	if paramsTooLong(q) {
		views.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	options, err := c.exams.Options(r.Context(), models.OptionsQuery{
		Mode:  q.Get("type"),
		Round: round,
		Date:  q.Get("date"),
		Time:  q.Get("time"),
	})
	if errors.Is(err, models.ErrInvalidOptionMode) {
		views.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Options query error: %v", err)
		views.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if len(options) == 0 {
		views.NotFound(w, "No options found")
		return
	}

	views.JSON(w, http.StatusOK, options)
}
