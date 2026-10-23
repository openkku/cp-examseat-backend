// Package admin implements the token-protected administration used by the
// frontend's admin page: importing seating files, renaming and deleting
// rounds and custom datasets, and reloading the in-memory state.
package admin

import "time"

// CustomDataset is an out-of-schedule dataset imported under a custom ID.
type CustomDataset struct {
	ID    string `json:"id"`
	Seats int    `json:"seats"`
}

// RoundSummary describes the data stored for one round.
type RoundSummary struct {
	ID              string          `json:"id"`
	Label           string          `json:"label"`
	Seats           int             `json:"seats"`
	Students        int             `json:"students"`
	InScheduleSeats int             `json:"in_schedule_seats"`
	CustomDatasets  []CustomDataset `json:"custom_datasets"`
}

// Status is the state shown on the admin page.
type Status struct {
	Rooms           int
	RoomsConfigured int
	LastReload      time.Time
}

// ImportRequest is one seating file to import.
type ImportRequest struct {
	FilePath   string
	Round      string
	Display    string
	Labels     []string
	RoomLayout string
	CustomID   string
}
