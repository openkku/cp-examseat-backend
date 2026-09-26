package controllers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maypok86/otter/v2"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// CalendarController serves per-student iCalendar subscription feeds.
type CalendarController struct {
	exams *services.ExamService
	// feeds caches rendered .ics files per normalized student ID,
	// retaining each for 10 minutes since last access.
	feeds *otter.Cache[string, []byte]
}

// NewCalendarController creates a CalendarController.
func NewCalendarController(exams *services.ExamService) *CalendarController {
	return &CalendarController{
		exams: exams,
		feeds: otter.Must(&otter.Options[string, []byte]{
			MaximumSize:      5000,
			ExpiryCalculator: otter.ExpiryAccessing[string, []byte](10 * time.Minute),
		}),
	}
}

// Show handles GET /api/calendar/{id} and /api/calendar/{id}.ics.
func (c *CalendarController) Show(w http.ResponseWriter, r *http.Request) {
	idParam := strings.TrimSuffix(chi.URLParam(r, "id"), ".ics")
	id := models.NormalizeStudentID(idParam)
	if id == "" {
		views.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	ics, ok := c.feeds.GetIfPresent(id)
	if !ok {
		seats, err := c.exams.StudentSchedule(r.Context(), id, nil)
		if err != nil {
			log.Printf("Error querying exams for calendar: %v", err)
			views.Error(w, "Error generating calendar feed", http.StatusInternalServerError)
			return
		}

		if len(seats) == 0 {
			views.NotFound(w, "No exam schedules found for this student")
			return
		}

		ics, err = views.RenderCalendar(seats)
		if err != nil {
			log.Printf("Error generating calendar: %v", err)
			views.Error(w, "Error generating calendar feed", http.StatusInternalServerError)
			return
		}

		c.feeds.Set(id, ics)
	}

	views.Calendar(w, "exams-"+idParam+".ics", ics)
}
