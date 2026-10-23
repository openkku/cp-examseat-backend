// Package scrapler is the client of the scrapler service
// (github.com/openkku/cp-examseat-scrapler), which runs next to the KKU VPN and
// does the actual paced lookups on the exam site
package scrapler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

const (
	maxBodyBytes = 1 << 20
	// requestTimeout must outlast scrapler's own wait (SCRAPLER_MAX_WAIT, 20s by default).
	requestTimeout = 30 * time.Second
)

// Client talks to one scrapler instance.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a Client for the scrapler at baseURL, e.g. "http://scrapler:8090".
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: requestTimeout}}
}

var _ checkex.Source = (*Client)(nil)

type lookupRequest struct {
	StudentID string `json:"student_id"`
	Client    string `json:"client"`
}

type lookupResponse struct {
	Status     string `json:"status"`
	Position   int    `json:"position"`
	RetryAfter int    `json:"retry_after"`
	Error      string `json:"error"`
	Seats      []struct {
		Sheet       string `json:"sheet"`
		Date        string `json:"date"`
		TimeStart   string `json:"time_start"`
		TimeEnd     string `json:"time_end"`
		Room        string `json:"room"`
		Subject     string `json:"subject"`
		SubjectName string `json:"subject_name"`
		Section     string `json:"section"`
		Seat        string `json:"seat"`
		Note        string `json:"note"`
		Branch      string `json:"branch"`
	} `json:"seats"`
}

// Lookup fetches one student's seats. found is false when checkex has no seats
// for the ID. client identifies the visitor for scrapler's per-client queue cap
// ("" is never capped).
//
// It returns *checkex.LookupQueuedError when the lookup is still waiting in
// scrapler's queue, checkex.ErrLookupLimited when the visitor has too many
// lookups queued, checkex.ErrLookupBusy when the queue is full and
// checkex.ErrUpstreamUnavailable when checkex can't be reached at all.
func (c *Client) Lookup(ctx context.Context, roundID, studentID, client string) ([]domain.Seat, bool, error) {
	body, err := json.Marshal(lookupRequest{StudentID: studentID, Client: client})
	if err != nil {
		return nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/lookup", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("scrapler: %w", err)
	}
	defer resp.Body.Close()

	var out lookupResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&out); err != nil {
		return nil, false, fmt.Errorf("scrapler status %d: %w", resp.StatusCode, err)
	}

	switch out.Status {
	case "found":
		seats := make([]domain.Seat, 0, len(out.Seats))
		for _, s := range out.Seats {
			seats = append(seats, domain.Seat{
				Sheet:       s.Sheet,
				Date:        s.Date,
				TimeStart:   s.TimeStart,
				TimeEnd:     s.TimeEnd,
				Category:    domain.CategoryInSchedule,
				Room:        s.Room,
				Subject:     s.Subject,
				SubjectName: s.SubjectName,
				Section:     s.Section,
				StudentID:   studentID,
				Seat:        s.Seat,
				Note:        s.Note,
				ExamRound:   roundID,
				Branch:      s.Branch,
			})
		}
		return seats, len(seats) > 0, nil
	case "not_found":
		return nil, false, nil
	case "queued":
		return nil, false, &checkex.LookupQueuedError{Position: out.Position, RetryAfter: out.RetryAfter}
	case "limited":
		return nil, false, checkex.ErrLookupLimited
	case "busy":
		return nil, false, checkex.ErrLookupBusy
	case "unavailable":
		return nil, false, checkex.ErrUpstreamUnavailable
	case "error", "invalid":
		return nil, false, fmt.Errorf("scrapler %s: %s", out.Status, out.Error)
	default:
		return nil, false, fmt.Errorf("scrapler status %d: unknown answer %q", resp.StatusCode, out.Status)
	}
}
