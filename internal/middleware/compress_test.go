package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klauspost/compress/zstd"

	"github.com/openkku/cp-examseat-backend/internal/compression"
)

func TestCompressMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hello": "world", "status": "ok", "message": "this is a test message to ensure it is longer than 512 bytes so that we trigger compression easily! Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."}`))
	})

	mw := Compress(handler)

	// Test with compression (zstd)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "zstd, br, gzip")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	res := rec.Result()
	if got := res.Header.Get("Content-Encoding"); got != "zstd" {
		t.Errorf("Content-Encoding = %q; want %q", got, "zstd")
	}
	if got := res.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q; want %q", got, "Accept-Encoding")
	}

	body, _ := io.ReadAll(res.Body)
	zr, err := zstd.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to init zstd reader: %v", err)
	}
	defer zr.Close()
	decompressed, _ := io.ReadAll(zr)

	expectedPrefix := `{"hello": "world"`
	if !bytes.HasPrefix(decompressed, []byte(expectedPrefix)) {
		t.Errorf("decompressed response content mismatch: got %q", string(decompressed))
	}
}

func TestCompressMiddlewareSkipNonCompressible(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake png content fake png content fake png content"))
	})

	mw := Compress(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	res := rec.Result()
	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding should be empty, got %q", got)
	}
}

func TestCompressMiddlewareExposesEncodingAndRespectsPrecompressed(t *testing.T) {
	var seen string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = compression.GetEncoding(r)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("already-compressed"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()
	Compress(handler).ServeHTTP(rec, req)

	if seen != "br" {
		t.Errorf("handler saw encoding %q; want br", seen)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d; want 404", rec.Code)
	}
	if rec.Body.String() != "already-compressed" {
		t.Errorf("body was re-encoded: %q", rec.Body.String())
	}
}
