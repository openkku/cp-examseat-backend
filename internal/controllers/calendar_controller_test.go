package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

type mockDB struct {
	models.ExamRepository
	seats []models.Seat
	err   error
}

func (m *mockDB) GetSeatsByID(ctx context.Context, opts models.SeatQuery) ([]models.Seat, error) {
	return m.seats, m.err
}

func newCalendarController(db models.ExamRepository) *CalendarController {
	return NewCalendarController(services.NewExamService(db, nil))
}

func TestHandleGetCalendarFeed_NoData(t *testing.T) {
	db := &mockDB{
		seats: []models.Seat{},
		err:   nil,
	}

	req := httptest.NewRequest("GET", "/api/calendar/123456789.ics", nil)
	w := httptest.NewRecorder()

	// Use chi context to pass route params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "123456789")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	newCalendarController(db).Show(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	var errResp views.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedError := "No exam schedules found for this student"
	if errResp.Error != expectedError {
		t.Errorf("expected error message %q, got %q", expectedError, errResp.Error)
	}
}

func TestHandleGetCalendarFeed_Success(t *testing.T) {
	db := &mockDB{
		seats: []models.Seat{
			{
				StudentID:   "123456789",
				Date:        "2026-06-10",
				Time:        "09.00 - 12.00",
				Subject:     "CP001001",
				SubjectName: "Introduction to Computer Science",
				Section:     "1",
				Room:        "Building 3 Room 301",
				Seat:        "A1",
				ExamRound:   "midterm",
			},
		},
		err: nil,
	}

	testCases := []struct {
		name             string
		idParamVal       string
		expectedFilename string
	}{
		{
			name:             "Plain ID",
			idParamVal:       "123456789",
			expectedFilename: "exams-123456789.ics",
		},
		{
			name:             "ID with .ics suffix",
			idParamVal:       "123456789.ics",
			expectedFilename: "exams-123456789.ics",
		},
		{
			name:             "ID with hyphen and suffix",
			idParamVal:       "12345678-9.ics",
			expectedFilename: "exams-12345678-9.ics",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/calendar/"+tc.idParamVal, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.idParamVal)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			newCalendarController(db).Show(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "text/calendar") {
				t.Errorf("expected Content-Type to start with 'text/calendar', got %q", contentType)
			}

			expectedDisposition := `attachment; filename="` + tc.expectedFilename + `"`
			contentDisposition := resp.Header.Get("Content-Disposition")
			if contentDisposition != expectedDisposition {
				t.Errorf("expected Content-Disposition %q, got %q", expectedDisposition, contentDisposition)
			}
		})
	}
}
