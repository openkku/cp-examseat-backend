package scrapler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

const (
	fakeID    = "6833800999"
	testRound = "final_1_2569"
)

func TestClientLookup(t *testing.T) {
	var got lookupRequest
	answer := `{"status":"found","seats":[{"sheet":"20 ต.ค. 69 (เช้า)","date":"2026-10-20","time_start":"08.30","time_end":"11.30","room":"SC.7401","subject":"CP422022","subject_name":"Database Architecture Analysis and Design","section":"02","seat":"B51","note":"","branch":"CP-Cy"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/lookup" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()

	seats, ok, err := NewClient(srv.URL+"/").Lookup(context.Background(), testRound, fakeID, "1.2.3.4")
	if err != nil || !ok || len(seats) != 1 {
		t.Fatalf("ok=%v err=%v seats=%d", ok, err, len(seats))
	}
	if got != (lookupRequest{StudentID: fakeID, Client: "1.2.3.4"}) {
		t.Fatalf("scrapler got %+v", got)
	}
	want := domain.Seat{
		Sheet: "20 ต.ค. 69 (เช้า)", Date: "2026-10-20", TimeStart: "08.30", TimeEnd: "11.30", Category: "IN_SCHEDULE",
		Room: "SC.7401", Subject: "CP422022", SubjectName: "Database Architecture Analysis and Design", Section: "02",
		StudentID: fakeID, Seat: "B51", ExamRound: testRound, Branch: "CP-Cy",
	}
	if g := seats[0]; fmt.Sprintf("%+v", g) != fmt.Sprintf("%+v", want) {
		t.Errorf("got  %+v\nwant %+v", g, want)
	}
}

func TestClientAnswers(t *testing.T) {
	cases := []struct {
		status int
		body   string
		check  func(error) bool
	}{
		{http.StatusOK, `{"status":"not_found"}`, func(err error) bool { return err == nil }},
		{http.StatusAccepted, `{"status":"queued","position":4}`, func(err error) bool {
			var q *checkex.LookupQueuedError
			return errors.As(err, &q) && q.Position == 4
		}},
		{http.StatusTooManyRequests, `{"status":"limited"}`, func(err error) bool { return errors.Is(err, checkex.ErrLookupLimited) }},
		{http.StatusServiceUnavailable, `{"status":"busy"}`, func(err error) bool { return errors.Is(err, checkex.ErrLookupBusy) }},
		{http.StatusBadGateway, `{"status":"error","error":"vpn down"}`, func(err error) bool { return err != nil }},
		{http.StatusBadGateway, `<html>proxy error</html>`, func(err error) bool { return err != nil }},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
			_, _ = w.Write([]byte(c.body))
		}))
		_, ok, err := NewClient(srv.URL).Lookup(context.Background(), testRound, fakeID, "")
		srv.Close()
		if ok || !c.check(err) {
			t.Errorf("%s: ok=%v err=%v", c.body, ok, err)
		}
	}
}
