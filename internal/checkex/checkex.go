// Package checkex fetches a student's seats on demand from checkex (through
// the scrapler service) for rounds that were never imported as files, and
// remembers every lookup so a student is fetched at most once.
package checkex

import "time"

// Upstream lookup outcomes, recorded per (round, student) so a student is
// fetched from checkex at most once.
const (
	LookupFound    = "found"
	LookupNotFound = "not_found"
)

// Lookup is the recorded outcome of one upstream (checkex) lookup.
type Lookup struct {
	Status    string
	FetchedAt time.Time
}

// Options configures on-demand lookups from checkex.
type Options struct {
	// Rounds served by checkex; empty disables the feature.
	Rounds []string
	// RoundLabels overrides the derived display label per round ID.
	RoundLabels map[string]string
	// IDPrefixes are the intake years whose 10-digit IDs may be sent upstream.
	IDPrefixes []string
	// NotFoundTTL is how long a "not found" answer is trusted.
	NotFoundTTL time.Duration
	// NotFoundLimit is the not-found answers one visitor may cause per
	// NotFoundWindow; 0 disables the limit.
	NotFoundLimit  int
	NotFoundWindow time.Duration
}
