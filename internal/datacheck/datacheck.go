package datacheck

// ProblemKind identifies the category of configuration or data problem.
type ProblemKind string

const (
	KindMissingMetadata ProblemKind = "missing_metadata"
	KindInvalidLayout   ProblemKind = "invalid_layout"
	KindMissingSeats    ProblemKind = "missing_seats"
	KindMissingImage    ProblemKind = "missing_image"
	KindUnusedRoom      ProblemKind = "unused_room"
)

// Problem is one discrepancy detected by health checks.
type Problem struct {
	Kind     ProblemKind `json:"kind"`
	Severity string      `json:"severity"` // "error", "warning", "info"
	Room     string      `json:"room,omitempty"`
	Round    string      `json:"round,omitempty"`
	Message  string      `json:"message"`
	Details  []string    `json:"details,omitempty"`
}

// Report collects all detected problems.
type Report struct {
	Problems []Problem `json:"problems"`
}
