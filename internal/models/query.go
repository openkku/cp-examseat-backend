package models

// Option modes accepted by OptionsQuery.Mode.
const (
	OptionDates = "dates"
	OptionTimes = "times"
	OptionRooms = "rooms"
)

// SeatQuery selects every seat of one student, optionally narrowed by round and sheet.
type SeatQuery struct {
	StudentID string
	Round     *string
	Sheet     *string
}

// ExploreQuery selects the roster of one room for one exam slot.
type ExploreQuery struct {
	Round string
	Room  string
	Date  string
	Time  string
	Seat  *string
}

// OptionsQuery lists the distinct dates, times or rooms of a round for the
// cascading explorer filters.
type OptionsQuery struct {
	Mode  string
	Round string
	Date  string
	Time  string
	// RoomFilter, when set, keeps only rows whose room it accepts
	// (used to hide rooms that have no seating layout).
	RoomFilter func(room string) bool
}

// AcceptsRoom reports whether a room passes the optional RoomFilter.
func (q OptionsQuery) AcceptsRoom(room string) bool {
	return q.RoomFilter == nil || q.RoomFilter(room)
}
