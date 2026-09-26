package views

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/maypok86/otter/v2"

	"github.com/openkku/cp-examseat-backend/internal/compression"
)

// ErrRender reports a failure to serialize or compress a response.
var ErrRender = errors.New("failed to render response")

// cachedResponse is a rendered JSON body, already compressed with the
// encoding that is part of its cache key.
type cachedResponse struct {
	status int
	body   []byte
}

// ResponseCache caches rendered, pre-compressed JSON responses so hot
// endpoints skip the database, serialization and compression entirely.
type ResponseCache struct {
	store *otter.Cache[string, cachedResponse]
}

// NewResponseCache creates a cache holding up to maxSize responses, each
// expiring ttl after its last access.
func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		store: otter.Must(&otter.Options[string, cachedResponse]{
			MaximumSize:      maxSize,
			ExpiryCalculator: otter.ExpiryAccessing[string, cachedResponse](ttl),
		}),
	}
}

// RenderFunc produces the status and payload of a response on a cache miss.
type RenderFunc func() (status int, payload any, err error)

// ServeJSON writes the cached response for key, calling render on a miss.
// Responses are cached per negotiated content encoding. An error from render
// is returned unchanged and nothing is written or cached; serialization or
// compression failures are returned wrapping ErrRender.
func (c *ResponseCache) ServeJSON(w http.ResponseWriter, r *http.Request, key string, render RenderFunc) error {
	encoding := compression.GetEncoding(r)
	cacheKey := key + ":" + encoding

	entry, ok := c.store.GetIfPresent(cacheKey)
	if !ok {
		status, payload, err := render()
		if err != nil {
			return err
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRender, err)
		}
		compressed, err := compression.Bytes(body, encoding)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRender, err)
		}
		entry = cachedResponse{status: status, body: compressed}
		c.store.Set(cacheKey, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	if encoding != "" {
		w.Header().Set("Content-Encoding", encoding)
		w.Header().Set("Vary", "Accept-Encoding")
	}
	w.WriteHeader(entry.status)
	w.Write(entry.body)
	return nil
}
