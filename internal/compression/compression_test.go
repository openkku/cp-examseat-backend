package compression

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func TestNegotiate(t *testing.T) {
	tests := []struct {
		acceptEncoding string
		expected       string
	}{
		{"gzip, deflate, br", "br"},                                 // br is preferred over gzip, deflate (equal q-value, sorted by server pref)
		{"gzip;q=0.5, deflate;q=0.5, br;q=0.4, zstd;q=0.8", "zstd"}, // zstd has highest q
		{"gzip;q=0.5, deflate;q=0.5, br;q=0.9, zstd;q=0.8", "br"},   // br has highest q
		{"gzip", "gzip"},
		{"deflate", "deflate"},
		{"zstd", "zstd"},
		{"*", "zstd"},         // default preferred
		{"*, br;q=0", "zstd"}, // br rejected
		{"identity", ""},
		{"randomencoding", ""},
		{"gzip;q=0, deflate", "deflate"},
		{"", ""},
	}

	for _, tc := range tests {
		got := Negotiate(tc.acceptEncoding)
		if got != tc.expected {
			t.Errorf("Negotiate(%q) = %q; want %q", tc.acceptEncoding, got, tc.expected)
		}
	}
}

func TestCompressBytes(t *testing.T) {
	input := []byte("hello world hello world hello world hello world hello world")

	encodings := []string{"zstd", "br", "gzip", "deflate"}
	for _, enc := range encodings {
		compressed, err := Bytes(input, enc)
		if err != nil {
			t.Fatalf("Bytes error for %s: %v", enc, err)
		}

		// Decompress and check
		var decompressed []byte
		switch enc {
		case "zstd":
			zr, err := zstd.NewReader(bytes.NewReader(compressed))
			if err != nil {
				t.Fatalf("failed to create zstd reader: %v", err)
			}
			decompressed, err = io.ReadAll(zr)
			if err != nil {
				t.Fatalf("failed to read zstd: %v", err)
			}
			zr.Close()
		case "br":
			br := brotli.NewReader(bytes.NewReader(compressed))
			decompressed, err = io.ReadAll(br)
			if err != nil {
				t.Fatalf("failed to read br: %v", err)
			}
		case "gzip":
			gr, err := gzip.NewReader(bytes.NewReader(compressed))
			if err != nil {
				t.Fatalf("failed to create gzip reader: %v", err)
			}
			decompressed, err = io.ReadAll(gr)
			if err != nil {
				t.Fatalf("failed to read gzip: %v", err)
			}
			gr.Close()
		case "deflate":
			fr := flate.NewReader(bytes.NewReader(compressed))
			decompressed, err = io.ReadAll(fr)
			if err != nil {
				t.Fatalf("failed to read deflate: %v", err)
			}
			fr.Close()
		}

		if string(decompressed) != string(input) {
			t.Errorf("decompressed content for %s mismatch: got %q; want %q", enc, string(decompressed), string(input))
		}
	}
}
