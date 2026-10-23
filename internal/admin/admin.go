// Package admin implements data management for operators: importing seating
// files, listing, renaming and deleting rounds and custom datasets, and asking
// the server to reload. It is driven by the operator app and the migrate CLI,
// never by the public site.
package admin

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

// ImportRequest is one seating file to import.
type ImportRequest struct {
	FilePath   string
	Round      string
	Display    string
	Labels     []string
	RoomLayout string
	CustomID   string
}

// ReplaceTarget describes what existing seats will be replaced by an import.
type ReplaceTarget struct {
	Type  string `json:"type"`  // "in_schedule" or "custom_dataset"
	ID    string `json:"id"`    // round ID or custom dataset ID
	Seats int    `json:"seats"` // current seat count
}

// ImportPreview summarizes the content and impact of a seating file before import.
type ImportPreview struct {
	Seats     int             `json:"seats"`
	Students  int             `json:"students"`
	Rooms     []string        `json:"rooms"`
	Dates     []string        `json:"dates"`
	Sessions  int             `json:"sessions"`
	CustomIDs []string        `json:"custom_ids"`
	Replaces  []ReplaceTarget `json:"replaces"`
	Rows      []PreviewRow    `json:"rows"`
}

// PreviewRow is one parsed seat shown in an import preview.
type PreviewRow struct {
	Sheet       string   `json:"sheet"`
	Date        string   `json:"date"`
	Time        string   `json:"time"`
	Room        string   `json:"room"`
	Subject     string   `json:"subject"`
	SubjectName string   `json:"subject_name"`
	Section     string   `json:"section"`
	StudentID   string   `json:"student_id"`
	Seat        string   `json:"seat"`
	CustomID    string   `json:"custom_id"`
	Labels      []string `json:"labels"`
}
