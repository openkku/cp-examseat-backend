package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maypok86/otter/v2"

	"github.com/openkku/cp-examseat-backend/internal/exam/presenter"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

var filenameSafe = regexp.MustCompile(`[^0-9-]+`)

// Feeds returns the seats of a student's calendar feed; complete is false when
// the feed must not be cached.
type Feeds interface {
	CalendarSeats(ctx context.Context, studentID, client string) (seats []domain.Seat, complete bool, err error)
}

// Calendar serves per-student iCalendar subscription feeds.
type Calendar struct {
	exams    Feeds
	clientIP httpx.ClientIPFunc
	// feeds caches rendered .ics files per normalized student ID,
	// retaining each for 10 minutes since last access.
	feeds *otter.Cache[string, []byte]
}

// NewCalendar creates a Calendar handler.
func NewCalendar(exams Feeds, clientIP httpx.ClientIPFunc) *Calendar {
	return &Calendar{
		exams:    exams,
		clientIP: clientIP,
		feeds: otter.Must(&otter.Options[string, []byte]{
			MaximumSize:      5000,
			ExpiryCalculator: otter.ExpiryAccessing[string, []byte](10 * time.Minute),
		}),
	}
}

// ClearCache drops every cached feed (after the underlying data changed).
func (h *Calendar) ClearCache() {
	h.feeds.InvalidateAll()
}

// Show handles GET /api/calendar/{id} and /api/calendar/{id}.ics.
func (h *Calendar) Show(w http.ResponseWriter, r *http.Request) {
	// chi returns path parameters still percent-encoded.
	rawID, err := url.PathUnescape(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}
	idParam := strings.TrimSuffix(rawID, ".ics")
	id := domain.NormalizeStudentID(idParam)
	if id == "" {
		httpx.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}
	if len(id) > httpx.MaxStudentIDDigits {
		httpx.Error(w, "Parameter too long", http.StatusBadRequest)
		return
	}

	ics, ok := h.feeds.GetIfPresent(id)
	if !ok {
		// Don't cache a feed built while an upstream fetch failed.
		seats, complete, err := h.exams.CalendarSeats(r.Context(), id, httpx.ClientIP(h.clientIP, r))
		if err != nil {
			log.Printf("Error querying exams for calendar: %v", err)
			httpx.Error(w, "Error generating calendar feed", http.StatusInternalServerError)
			return
		}

		if len(seats) == 0 {
			httpx.NotFound(w, "No exam schedules found for this student")
			return
		}

		ics, err = presenter.RenderCalendar(seats)
		if err != nil {
			log.Printf("Error generating calendar: %v", err)
			httpx.Error(w, "Error generating calendar feed", http.StatusInternalServerError)
			return
		}

		if complete {
			h.feeds.Set(id, ics)
		}
	}

	// Only digits and dashes reach the Content-Disposition filename.
	writeCalendar(w, "exams-"+filenameSafe.ReplaceAllString(idParam, "")+".ics", ics)
}

// writeCalendar writes an iCalendar feed as a downloadable attachment.
func writeCalendar(w http.ResponseWriter, filename string, ics []byte) {
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.Write(ics)
}
