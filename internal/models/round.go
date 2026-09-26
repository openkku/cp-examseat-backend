package models

import (
	"fmt"
	"strconv"
	"strings"
)

// Round is an exam round such as "mid_1_2569" (midterm, semester 1, B.E. 2569).
type Round struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// parseRoundID parses a round ID string into year, semester, and a type weight
func parseRoundID(id string) (year int, semester int, typeWeight int) {
	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		return 0, 0, 0
	}

	if y, err := strconv.Atoi(parts[2]); err == nil {
		year = y
	}

	if sem, err := strconv.Atoi(parts[1]); err == nil {
		semester = sem
	}

	// type weight: final = 2, mid = 1, others = 0
	switch parts[0] {
	case "final":
		typeWeight = 2
	case "mid":
		typeWeight = 1
	}

	return
}

// CompareRounds returns true if idA should come before idB in descending chronological order
func CompareRounds(idA, idB string) bool {
	yearA, semA, typeA := parseRoundID(idA)
	yearB, semB, typeB := parseRoundID(idB)

	if yearA != yearB {
		return yearA > yearB
	}
	if semA != semB {
		return semA > semB
	}
	if typeA != typeB {
		return typeA > typeB
	}
	return idA > idB
}

// DefaultRoundLabel builds a Thai display label from a round ID,
// e.g. "mid_1_2569" -> "กลางภาค 1/2569". Unknown formats are returned as-is.
func DefaultRoundLabel(roundID string) string {
	parts := strings.Split(roundID, "_")
	if len(parts) == 3 {
		term, semester, year := parts[0], parts[1], parts[2]
		switch term {
		case "mid":
			return fmt.Sprintf("กลางภาค %s/%s", semester, year)
		case "final":
			return fmt.Sprintf("ปลายภาค %s/%s", semester, year)
		}
	}
	return roundID
}

// CustomDataset is an out-of-schedule dataset imported under a custom ID.
type CustomDataset struct {
	ID    string `json:"id"`
	Seats int    `json:"seats"`
}

// RoundSummary describes the data stored for one round (admin view).
type RoundSummary struct {
	ID              string          `json:"id"`
	Label           string          `json:"label"`
	Seats           int             `json:"seats"`
	Students        int             `json:"students"`
	InScheduleSeats int             `json:"in_schedule_seats"`
	CustomDatasets  []CustomDataset `json:"custom_datasets"`
}
