package roomconfig

import (
	"encoding/json"
	"fmt"
)

// Layout is the seating layout structure stored in room/map/*.json.
type Layout struct {
	FrontLabel string        `json:"frontLabel,omitempty"`
	BackLabel  string        `json:"backLabel,omitempty"`
	Layout     []LayoutBlock `json:"layout"`
}

// LayoutBlock is one column or aisle in the layout.
type LayoutBlock struct {
	Type  string       `json:"type"` // "column" or "aisle"
	Label string       `json:"label,omitempty"`
	Items []LayoutItem `json:"items,omitempty"`
	Width int          `json:"width,omitempty"`
}

// LayoutItem is one element inside a column.
type LayoutItem struct {
	Type        string `json:"type"` // "seats", "obstruction" or "gap"
	Char        string `json:"char,omitempty"`
	Start       *int   `json:"start,omitempty"`
	Count       int    `json:"count,omitempty"`
	Step        *int   `json:"step,omitempty"`
	Inverse     bool   `json:"inverse,omitempty"`
	Manual      []int  `json:"manual,omitempty"`
	Label       string `json:"label,omitempty"`
	Height      int    `json:"height,omitempty"`
	Width       int    `json:"width,omitempty"`
	Transparent bool   `json:"transparent,omitempty"`
}

// ParseLayout decodes JSON data into a Layout.
func ParseLayout(data []byte) (Layout, error) {
	var l Layout
	if err := json.Unmarshal(data, &l); err != nil {
		return Layout{}, fmt.Errorf("%w: %v", ErrInvalidLayout, err)
	}
	return l, nil
}

// SeatIDs extracts all seat identifiers in order across the layout according to
// the site's SeatMap numbering rules.
func (l Layout) SeatIDs() []string {
	counters := make(map[string]int)
	var seats []string

	for _, block := range l.Layout {
		if block.Type != "column" {
			continue
		}

		for _, item := range block.Items {
			if item.Type != "seats" {
				continue
			}

			// Like the site, a manual list (even an empty one) replaces counting.
			if item.Manual != nil {
				nums := make([]int, len(item.Manual))
				copy(nums, item.Manual)
				if item.Inverse {
					for i, j := 0, len(nums)-1; i < j; i, j = i+1, j-1 {
						nums[i], nums[j] = nums[j], nums[i]
					}
				}
				for _, num := range nums {
					seats = append(seats, fmt.Sprintf("%s%d", item.Char, num))
				}
			} else {
				start := 1
				// JS `counter || 1`: 0 starts at 1, a negative counter is kept.
				if current := counters[item.Char]; current != 0 {
					start = current
				}
				if item.Start != nil {
					start = *item.Start
				}

				step := 1
				if item.Inverse {
					step = -1
				}
				if item.Step != nil {
					step = *item.Step
				}

				current := start
				for i := 0; i < item.Count; i++ {
					seats = append(seats, fmt.Sprintf("%s%d", item.Char, current))
					current += step
				}
				counters[item.Char] = current
			}
		}
	}

	return seats
}
