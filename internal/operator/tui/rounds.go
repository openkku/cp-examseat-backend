package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

func (m Model) updateRounds(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeDatasets {
		return m.updateDatasets(key)
	}
	n := len(m.roundList)
	switch key.String() {
	case "up", "k":
		m.roundCursor = clamp(m.roundCursor-1, n)
	case "down", "j":
		m.roundCursor = clamp(m.roundCursor+1, n)
	case "i":
		m.openForm(m.importForm())
	case "r":
		if r, ok := m.selectedRound(); ok {
			m.openForm(m.renameForm(r))
		}
	case "d":
		if r, ok := m.selectedRound(); ok {
			m.ask(fmt.Sprintf("Delete round %q (%s) and all %d seats? This cannot be undone.", r.Label, r.ID, r.Seats),
				change("Deleted round "+r.ID, true, false, func() error { return m.rounds.DeleteRound(context.Background(), r.ID, "") }))
		}
	case "enter":
		if r, ok := m.selectedRound(); ok && len(r.CustomDatasets) > 0 {
			m.mode, m.datasetCursor = modeDatasets, 0
		}
	}
	return m, nil
}

func (m Model) updateDatasets(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	r, ok := m.selectedRound()
	if !ok {
		m.mode = modeList
		return m, nil
	}
	n := len(r.CustomDatasets)
	switch key.String() {
	case "esc", "backspace":
		m.mode = modeList
	case "up", "k":
		m.datasetCursor = clamp(m.datasetCursor-1, n)
	case "down", "j":
		m.datasetCursor = clamp(m.datasetCursor+1, n)
	case "d":
		if n == 0 {
			return m, nil
		}
		d := r.CustomDatasets[clamp(m.datasetCursor, n)]
		m.ask(fmt.Sprintf("Delete dataset %s (%d seats) from round %s?", d.ID, d.Seats, r.ID),
			change("Deleted dataset "+d.ID, true, false, func() error { return m.rounds.DeleteRound(context.Background(), r.ID, d.ID) }))
	}
	return m, nil
}

func (m Model) selectedRound() (admin.RoundSummary, bool) {
	if len(m.roundList) == 0 {
		return admin.RoundSummary{}, false
	}
	return m.roundList[clamp(m.roundCursor, len(m.roundList))], true
}

func (m Model) renameForm(r admin.RoundSummary) *form {
	return newForm("Rename "+r.ID, func(v []string) (tea.Cmd, error) {
		if v[0] == "" {
			return nil, errors.New("the label is required")
		}
		return change("Renamed "+r.ID, true, false, func() error {
			_, err := m.rounds.RenameRound(context.Background(), r.ID, v[0])
			return err
		}), nil
	}, newField("Label", r.Label, "กลางภาค 1/2569"))
}

func (m Model) importForm() *form {
	start := m.sourceDir
	if start != "" && !strings.HasSuffix(start, "/") {
		start += "/"
	}
	return newForm("Import a seating file", func(v []string) (tea.Cmd, error) {
		path, round, display, labels, layout, customID := v[0], v[1], v[2], v[3], v[4], v[5]
		if path == "" || strings.HasSuffix(path, "/") {
			return nil, errors.New("choose a file")
		}
		if err := m.rounds.ValidateFileType(strings.ToLower(filepath.Ext(path))); err != nil {
			return nil, err
		}
		if round == "" {
			return nil, errors.New("the round ID is required")
		}
		req := admin.ImportRequest{
			FilePath:   path,
			Round:      round,
			Display:    display,
			Labels:     domain.ParseLabels(labels),
			RoomLayout: layout,
			CustomID:   customID,
		}
		return func() tea.Msg {
			saved, err := m.rounds.Import(context.Background(), req)
			return doneMsg{text: fmt.Sprintf("Imported %d seats into %s", saved, round), err: err, reloadRounds: true}
		}, nil
	},
		newPathField("File (on this server)", start),
		newField("Round ID", "", "mid_1_2569"),
		newField("Display name (optional)", "", "กลางภาค 1/2569"),
		newField("Labels, comma-separated (custom datasets)", "", "LAB,Lab"),
		newField("Room layout (optional)", "", "CP9421_LAB"),
		newField("Custom ID (optional; replaces only this dataset)", "", "FINAL_2_OOP_LAB_2026"),
	)
}

func (m Model) roundsView() string {
	var b strings.Builder
	if m.mode == modeDatasets {
		r, _ := m.selectedRound()
		b.WriteString(titleStyle.Render("Custom datasets of "+r.ID) + "\n\n")
		for i, d := range r.CustomDatasets {
			line := fmt.Sprintf("%-32s %6d seats", d.ID, d.Seats)
			b.WriteString(cursorLine(i == m.datasetCursor, line) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑↓ select · d delete dataset · esc back"))
		return b.String()
	}

	b.WriteString(titleStyle.Render("Rounds") + "\n\n")
	if len(m.roundList) == 0 {
		b.WriteString(dimStyle.Render("No rounds yet. Press i to import a seating file.") + "\n")
	}
	for i, r := range m.roundList {
		line := fmt.Sprintf("%-24s %-28s %7d seats %6d students", r.ID, truncate(r.Label, 28), r.Seats, r.Students)
		if n := len(r.CustomDatasets); n > 0 {
			line += fmt.Sprintf("  +%d datasets", n)
		}
		b.WriteString(cursorLine(i == m.roundCursor, line) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("↑↓ select · i import · r rename · d delete · enter datasets · u reload site · ←→ tabs · q quit"))
	return b.String()
}

func cursorLine(selected bool, line string) string {
	if selected {
		return selectedStyle.Render("› " + line)
	}
	return "  " + line
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
