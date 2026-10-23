package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type stubSchedules struct{ err error }

func (s stubSchedules) StudentSchedule(context.Context, string, string, string) ([]domain.Seat, error) {
	return nil, s.err
}

func showExam(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/exam?id=6833809314&round=final_1_2569", nil)
	w := httptest.NewRecorder()
	NewExam(stubSchedules{err: err}, nil).Show(w, req)
	return w
}

func TestShowQueuedUsesScraplersRetryAfter(t *testing.T) {
	cases := []struct {
		err  *checkex.LookupQueuedError
		want int
	}{
		{&checkex.LookupQueuedError{Position: 2, RetryAfter: 4}, 4},
		// An older scrapler doesn't say; keep the default.
		{&checkex.LookupQueuedError{Position: 2}, queuedRetryAfter},
	}
	for _, c := range cases {
		w := showExam(t, c.err)
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", w.Code)
		}
		var body QueuedResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.RetryAfter != c.want || body.Position != 2 || !body.Queued {
			t.Errorf("body = %+v, want retry_after %d", body, c.want)
		}
		if got := w.Header().Get("Retry-After"); got != strconv.Itoa(c.want) {
			t.Errorf("Retry-After = %q, want %d", got, c.want)
		}
	}
}

func TestShowUnavailableSaysTheFacultySystemCannotBeReached(t *testing.T) {
	w := showExam(t, checkex.ErrUpstreamUnavailable)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ไม่สามารถติดต่อระบบ Checkex") {
		t.Errorf("body = %s", w.Body.String())
	}
}
