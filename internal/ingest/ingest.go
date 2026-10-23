// Package ingest imports seating files (.pdf, .xlsx/.xls, .json) into the
// database, replacing the round or custom datasets they contain.
package ingest

// Options describes one seating file to import.
type Options struct {
	FilePath    string
	RoundID     string
	DisplayName string
	Labels      []string
	RoomLayout  string
	CustomID    string
}

// Params are the defaults an Extractor applies to the seats it reads.
type Params struct {
	RoundID    string
	Labels     []string
	RoomLayout string
	CustomID   string
}
