// Package middleware holds cross-cutting HTTP concerns applied by the router.
package middleware

import (
	"compress/flate"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"

	"github.com/openkku/cp-examseat-backend/internal/compression"
)

// Compress negotiates content compression and wraps response writers accordingly.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding := compression.Negotiate(r.Header.Get("Accept-Encoding"))

		// Inject the negotiated encoding into request context for handler use.
		r = r.WithContext(compression.WithEncoding(r.Context(), encoding))

		if encoding == "" {
			next.ServeHTTP(w, r)
			return
		}

		cw := &compressResponseWriter{
			ResponseWriter: w,
			encoding:       encoding,
		}
		defer cw.Close()

		next.ServeHTTP(cw, r)
	})
}

type compressResponseWriter struct {
	http.ResponseWriter
	w             io.WriteCloser
	encoding      string
	statusCode    int
	headerWritten bool
}

func (cw *compressResponseWriter) WriteHeader(code int) {
	if cw.headerWritten {
		return
	}
	cw.statusCode = code
}

func (cw *compressResponseWriter) Write(b []byte) (int, error) {
	if !cw.headerWritten {
		cw.initCompression(b)
	}
	if cw.w != nil {
		return cw.w.Write(b)
	}
	return cw.ResponseWriter.Write(b)
}

func (cw *compressResponseWriter) initCompression(firstChunk []byte) {
	cw.headerWritten = true

	// If handler already compressed response (e.g. pre-compressed cache), skip compression.
	if cw.Header().Get("Content-Encoding") != "" {
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		}
		return
	}

	contentType := cw.Header().Get("Content-Type")
	if contentType == "" && len(firstChunk) > 0 {
		contentType = http.DetectContentType(firstChunk)
		cw.Header().Set("Content-Type", contentType)
	}

	if !isCompressible(contentType) {
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		}
		return
	}

	if clStr := cw.Header().Get("Content-Length"); clStr != "" {
		if cl, err := strconv.Atoi(clStr); err == nil && cl < 512 {
			if cw.statusCode != 0 {
				cw.ResponseWriter.WriteHeader(cw.statusCode)
			}
			return
		}
	}

	cw.Header().Set("Vary", "Accept-Encoding")

	switch cw.encoding {
	case "zstd":
		cw.Header().Set("Content-Encoding", "zstd")
		cw.Header().Del("Content-Length")
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		} else {
			cw.ResponseWriter.WriteHeader(http.StatusOK)
		}
		zw, _ := zstd.NewWriter(cw.ResponseWriter, zstd.WithEncoderLevel(zstd.SpeedFastest))
		cw.w = zw
	case "br":
		cw.Header().Set("Content-Encoding", "br")
		cw.Header().Del("Content-Length")
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		} else {
			cw.ResponseWriter.WriteHeader(http.StatusOK)
		}
		bw := brotli.NewWriterLevel(cw.ResponseWriter, 4)
		cw.w = bw
	case "gzip":
		cw.Header().Set("Content-Encoding", "gzip")
		cw.Header().Del("Content-Length")
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		} else {
			cw.ResponseWriter.WriteHeader(http.StatusOK)
		}
		gw, _ := gzip.NewWriterLevel(cw.ResponseWriter, gzip.BestSpeed)
		cw.w = gw
	case "deflate":
		cw.Header().Set("Content-Encoding", "deflate")
		cw.Header().Del("Content-Length")
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		} else {
			cw.ResponseWriter.WriteHeader(http.StatusOK)
		}
		fw, _ := flate.NewWriter(cw.ResponseWriter, flate.BestSpeed)
		cw.w = fw
	default:
		if cw.statusCode != 0 {
			cw.ResponseWriter.WriteHeader(cw.statusCode)
		}
	}
}

func (cw *compressResponseWriter) Close() error {
	if !cw.headerWritten {
		cw.initCompression(nil)
	}
	if cw.w != nil {
		return cw.w.Close()
	}
	return nil
}

func (cw *compressResponseWriter) Flush() {
	if !cw.headerWritten {
		cw.initCompression(nil)
	}
	if f, ok := cw.w.(interface{ Flush() error }); ok {
		f.Flush()
	} else if f, ok := cw.w.(interface{ Flush() }); ok {
		f.Flush()
	}
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func isCompressible(contentType string) bool {
	if contentType == "" {
		return false
	}
	parts := strings.Split(contentType, ";")
	ct := strings.TrimSpace(strings.ToLower(parts[0]))

	if strings.HasPrefix(ct, "text/") {
		return true
	}

	switch ct {
	case "application/json",
		"application/javascript",
		"application/x-javascript",
		"application/xml",
		"application/atom+xml",
		"application/rss+xml",
		"image/svg+xml",
		"image/bmp",
		"image/x-icon",
		"application/x-font-ttf",
		"font/opentype",
		"application/vnd.ms-fontobject",
		"application/font-woff",
		"application/font-woff2":
		return true
	}

	return false
}
