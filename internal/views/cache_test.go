package views

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"

	"github.com/openkku/cp-examseat-backend/internal/compression"
)

func TestResponseCacheServeJSON(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)
	calls := 0
	render := func() (int, any, error) {
		calls++
		return http.StatusNotFound, ErrorResponse{Error: "no exams found"}, nil
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/explore", nil)
		req = req.WithContext(compression.WithEncoding(req.Context(), "gzip"))
		rec := httptest.NewRecorder()

		if err := cache.ServeJSON(rec, req, "explore:x", render); err != nil {
			t.Fatalf("ServeJSON: %v", err)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d; want 404", rec.Code)
		}
		if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
			t.Errorf("Content-Encoding = %q; want gzip", got)
		}
		zr, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		body, _ := io.ReadAll(zr)
		if string(body) != `{"error":"no exams found"}` {
			t.Errorf("body = %s", body)
		}
	}
	if calls != 1 {
		t.Errorf("render called %d times; want 1 (second request should hit the cache)", calls)
	}

	// A different encoding is a different cache entry.
	rec := httptest.NewRecorder()
	if err := cache.ServeJSON(rec, httptest.NewRequest("GET", "/", nil), "explore:x", render); err != nil {
		t.Fatalf("ServeJSON: %v", err)
	}
	if calls != 2 || rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("identity request should render uncompressed; calls=%d enc=%q", calls, rec.Header().Get("Content-Encoding"))
	}
}

func TestResponseCacheDoesNotCacheErrors(t *testing.T) {
	cache := NewResponseCache(10, time.Minute)
	boom := errors.New("db down")
	calls := 0
	render := func() (int, any, error) { calls++; return 0, nil, boom }

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		if err := cache.ServeJSON(rec, httptest.NewRequest("GET", "/", nil), "k", render); !errors.Is(err, boom) {
			t.Fatalf("err = %v; want %v", err, boom)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("nothing should be written on error, got %q", rec.Body.String())
		}
	}
	if calls != 2 {
		t.Errorf("render called %d times; errors must not be cached", calls)
	}
}
