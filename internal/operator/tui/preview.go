package tui

import (
	"fmt"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

const cellWidth = 6

// renderPreview draws a seating layout as text: columns side by side, seats
// numbered exactly like the site's SeatMap (per-letter counters running
// across columns, start/step/inverse/manual).
func renderPreview(data []byte) (string, error) {
	lf, err := roomconfig.ParseLayout(data)
	if err != nil {
		return "", err
	}

	counters := map[string]int{}
	var columns [][]string
	var widths []int
	for _, block := range lf.Layout {
		if block.Type == "aisle" {
			w := max(1, block.Width/10)
			columns = append(columns, nil)
			widths = append(widths, w)
			continue
		}
		cells := []string{block.Label}
		for _, item := range block.Items {
			cells = append(cells, itemCells(item, counters)...)
		}
		columns = append(columns, cells)
		widths = append(widths, cellWidth)
	}

	rows := 0
	for _, c := range columns {
		rows = max(rows, len(c))
	}
	total := 0
	for _, w := range widths {
		total += w
	}

	var b strings.Builder
	front := lf.FrontLabel
	if front == "" {
		front = "Front"
	}
	back := lf.BackLabel
	if back == "" {
		back = "Back"
	}
	b.WriteString(center("▲ "+front, total) + "\n")
	for r := 0; r < rows; r++ {
		var line strings.Builder
		for i, c := range columns {
			cell := ""
			if r < len(c) {
				cell = c[r]
			}
			line.WriteString(pad(cell, widths[i]))
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
	}
	b.WriteString(center("▼ "+back, total))
	return b.String(), nil
}

func itemCells(item roomconfig.LayoutItem, counters map[string]int) []string {
	switch item.Type {
	case "seats":
		var numbers []int
		if item.Manual != nil {
			numbers = append(numbers, item.Manual...)
			if item.Inverse {
				for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
					numbers[i], numbers[j] = numbers[j], numbers[i]
				}
			}
		} else {
			current := counters[item.Char]
			if current == 0 {
				current = 1
			}
			if item.Start != nil {
				current = *item.Start
			}
			step := 1
			if item.Inverse {
				step = -1
			}
			if item.Step != nil {
				step = *item.Step
			}
			for i := 0; i < item.Count; i++ {
				numbers = append(numbers, current)
				current += step
			}
			counters[item.Char] = current
		}
		cells := make([]string, len(numbers))
		for i, n := range numbers {
			cells[i] = fmt.Sprintf("%s%d", item.Char, n)
		}
		return cells
	case "obstruction":
		n := max(1, item.Count, item.Height)
		cells := make([]string, n)
		for i := range cells {
			cells[i] = "▓▓▓▓"
		}
		if item.Label != "" {
			cells[0] = item.Label
		}
		return cells
	default: // gap
		return make([]string, max(1, item.Count))
	}
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w-1]) + " "
	}
	return s + strings.Repeat(" ", w-len(r))
}

func center(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	return strings.Repeat(" ", (w-n)/2) + s
}
