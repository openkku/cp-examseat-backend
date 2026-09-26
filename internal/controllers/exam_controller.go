package controllers

import (
	"net/http"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// ExamController serves a student's exam schedule.
type ExamController struct {
	exams *services.ExamService
}

// NewExamController creates an ExamController.
func NewExamController(exams *services.ExamService) *ExamController {
	return &ExamController{exams: exams}
}

// Show handles GET /api/exam?id={studentID}&round={roundID}.
func (c *ExamController) Show(w http.ResponseWriter, r *http.Request) {
	id := models.NormalizeStudentID(r.URL.Query().Get("id"))
	round := r.URL.Query().Get("round")

	if id == "" || round == "" {
		views.Error(w, "Student ID and Round are required", http.StatusBadRequest)
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
