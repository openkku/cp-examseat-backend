package checkex

import (
	"errors"
	"fmt"
)

// ErrLookupBusy means the scrapler queue is full; the visitor should retry later.
var ErrLookupBusy = errors.New("checkex: lookup queue is full")

// ErrUpstreamUnavailable means checkex can't be reached right now (with a
// scrapler hub: no agent is healthy), so the lookup was not queued at all.
var ErrUpstreamUnavailable = errors.New("checkex: upstream unavailable")

// ErrLookupLimited means this visitor has caused too many not-found lookups, or
// has too many lookups waiting in the scrapler queue. Nothing was sent upstream.
var ErrLookupLimited = errors.New("checkex: too many lookups from this client")

// LookupQueuedError means scrapler didn't finish the lookup in time. It keeps
// running there and its answer is kept for a while, so retrying later is served
// without a new upstream call.
type LookupQueuedError struct {
	// Position is how many lookups are ahead of this one; 0 means it is running now.
	Position int
	// RetryAfter is when to ask again, in seconds, as scrapler suggests it;
	// 0 when scrapler didn't say.
	RetryAfter int
}

func (e *LookupQueuedError) Error() string {
	return fmt.Sprintf("checkex: lookup queued (%d ahead)", e.Position)
}
