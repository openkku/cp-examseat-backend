package checkex

import (
	"errors"
	"fmt"
)

// ErrLookupBusy means the scrapler queue is full; the visitor should retry later.
var ErrLookupBusy = errors.New("checkex: lookup queue is full")

// ErrLookupLimited means this visitor has caused too many not-found lookups, or
// has too many lookups waiting in the scrapler queue. Nothing was sent upstream.
var ErrLookupLimited = errors.New("checkex: too many lookups from this client")

// LookupQueuedError means scrapler didn't finish the lookup in time. It keeps
// running there and its answer is kept for a while, so retrying later is served
// without a new upstream call.
type LookupQueuedError struct {
	// Position is how many lookups are ahead of this one; 0 means it is running now.
	Position int
}

func (e *LookupQueuedError) Error() string {
	return fmt.Sprintf("checkex: lookup queued (%d ahead)", e.Position)
}
