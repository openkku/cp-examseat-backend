package controllers

import (
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maypok86/otter/v2"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

var filenameSafe = regexp.MustCompile(`[^0-9-]+`)

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

// ClearCache drops every cached feed (after the underlying data changed).
func (c *CalendarController) ClearCache() {
	c.feeds.InvalidateAll()
}

// Show handles GET /api/calendar/{id} and /api/calendar/{id}.ics.
func (c *CalendarController) Show(w http.ResponseWriter, r *http.Request) {
	// chi returns path parameters still percent-encoded.
	rawID, err := url.PathUnescape(chi.URLParam(r, "id"))
	if err != nil {
		views.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}
	idParam := strings.TrimSuffix(rawID, ".ics")
	id := models.NormalizeStudentID(idParam)
	if id == "" {
		views.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}
	if len(id) > maxStudentIDDigits {
		views.Error(w, "Parameter too long", http.StatusBadRequest)
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

	// Only digits and dashes reach the Content-Disposition filename.
	views.Calendar(w, "exams-"+filenameSafe.ReplaceAllString(idParam, "")+".ics", ics)
}
