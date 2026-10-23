// Package compression implements HTTP content-encoding negotiation and the
// zstd/br/gzip/deflate codecs shared by the compress middleware and the views.
package compression

import (
	"bytes"
	"compress/flate"
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

type contextKey string

const encodingContextKey = contextKey("encoding")

// WithEncoding stores the negotiated encoding in the request context.
func WithEncoding(ctx context.Context, encoding string) context.Context {
	return context.WithValue(ctx, encodingContextKey, encoding)
}

// GetEncoding retrieves the negotiated compression encoding from the request context.
func GetEncoding(r *http.Request) string {
	if val, ok := r.Context().Value(encodingContextKey).(string); ok {
		return val
	}
	return ""
}

// Bytes compresses the input bytes using the specified encoding.
func Bytes(input []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "zstd":
		var buf bytes.Buffer
		zw, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
		if err != nil {
			return nil, err
		}
		if _, err := zw.Write(input); err != nil {
			zw.Close()
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "br":
		var buf bytes.Buffer
		bw := brotli.NewWriterLevel(&buf, 4)
		if _, err := bw.Write(input); err != nil {
			bw.Close()
			return nil, err
		}
		if err := bw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "gzip":
		var buf bytes.Buffer
		gw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
		if err != nil {
			return nil, err
		}
		if _, err := gw.Write(input); err != nil {
			gw.Close()
			return nil, err
		}
		if err := gw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "deflate":
		var buf bytes.Buffer
		fw, err := flate.NewWriter(&buf, flate.BestSpeed)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(input); err != nil {
			fw.Close()
			return nil, err
		}
		if err := fw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		return input, nil
	}
}

func serverPreference(name string) int {
	switch name {
	case "zstd":
		return 4
	case "br":
		return 3
	case "gzip":
		return 2
	case "deflate":
		return 1
	default:
		return 0
	}
}

// Negotiate picks the preferred supported encoding for an Accept-Encoding header
// (zstd > br > gzip > deflate on equal q-values). It returns "" for identity.
func Negotiate(acceptEncoding string) string {
	if acceptEncoding == "" {
		return ""
	}

	type spec struct {
		name string
		q    float64
	}

	var specs []spec
	parts := strings.Split(acceptEncoding, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name := part
		q := 1.0

		if idx := strings.Index(part, ";"); idx != -1 {
			name = strings.TrimSpace(part[:idx])
			qpart := strings.TrimSpace(part[idx+1:])
			if strings.HasPrefix(qpart, "q=") {
				if val, err := strconv.ParseFloat(qpart[2:], 64); err == nil {
					q = val
				}
			}
		}

		name = strings.ToLower(name)
		specs = append(specs, spec{name: name, q: q})
	}

	sort.Slice(specs, func(i, j int) bool {
		if specs[i].q != specs[j].q {
			return specs[i].q > specs[j].q
		}
		return serverPreference(specs[i].name) > serverPreference(specs[j].name)
	})

	for _, s := range specs {
		if s.q == 0 {
			continue
		}
		switch s.name {
		case "zstd":
			return "zstd"
		case "br":
			return "br"
		case "gzip":
			return "gzip"
		case "deflate":
			return "deflate"
		case "*":
			rejected := make(map[string]bool)
			for _, other := range specs {
				if other.q == 0 {
					rejected[other.name] = true
				}
			}
			if !rejected["zstd"] {
				return "zstd"
			}
			if !rejected["br"] {
				return "br"
			}
			if !rejected["gzip"] {
				return "gzip"
			}
			if !rejected["deflate"] {
				return "deflate"
			}
		}
	}

	return ""
}
