package tui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]`)

func (m Model) updateRooms(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.roomIDs)
	switch key.String() {
	case "up", "k":
		m.roomCursor = clamp(m.roomCursor-1, n)
	case "down", "j":
		m.roomCursor = clamp(m.roomCursor+1, n)
	case "/":
		m.searching = true
		return m, m.search.Focus()
	case "n":
		m.openForm(m.metaForm("", room.Meta{}))
	case "enter":
		if id, ok := m.selectedRoom(); ok {
			m.openForm(m.metaForm(id, m.roomMap[id]))
		}
	case "d":
		if id, ok := m.selectedRoom(); ok {
			m.ask(fmt.Sprintf("Delete room %s? Its layout file is removed too unless another room uses it.", id),
				change("Deleted room "+id, false, true, func() error { return m.rooms.DeleteRoom(id) }))
		}
	case "e":
		if id, ok := m.selectedRoom(); ok {
			return m.startLayoutEdit(id)
		}
	case "p":
		if id, ok := m.selectedRoom(); ok {
			data, err := m.rooms.Layout(m.roomMap[id].LayoutFile)
			if err == nil {
				var preview string
				if preview, err = renderPreview(data); err == nil {
					m.preview, m.mode = titleStyle.Render(id+" · "+m.roomMap[id].LayoutFile)+"\n\n"+preview, modePreview
					return m, nil
				}
			}
			m.setStatus("", err)
		}
	}
	return m, nil
}

func (m Model) updateSearch(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.search.SetValue("")
		fallthrough
	case "enter":
		m.searching = false
		m.search.Blur()
		m.refilter()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(key)
	m.refilter()
	return m, cmd
}

// refilter recomputes the visible rooms from roomMap and the search text.
func (m *Model) refilter() {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.roomIDs = m.roomIDs[:0]
	for id, meta := range m.roomMap {
		if q == "" || strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(meta.LayoutFile), q) {
			m.roomIDs = append(m.roomIDs, id)
		}
	}
	sort.Strings(m.roomIDs)
	m.roomCursor = clamp(m.roomCursor, len(m.roomIDs))
}

func (m Model) selectedRoom() (string, bool) {
	if len(m.roomIDs) == 0 {
		return "", false
	}
	return m.roomIDs[clamp(m.roomCursor, len(m.roomIDs))], true
}

// metaForm edits a room's metadata; an empty id adds a new room.
func (m Model) metaForm(id string, meta room.Meta) *form {
	latLng := ""
	if meta.Lat != nil && meta.Lng != nil {
		latLng = fmt.Sprintf("%g, %g", *meta.Lat, *meta.Lng)
	}
	fields := []field{
		newField("Layout file (room/map/)", meta.LayoutFile, "SC1102.json"),
		newField("Blueprint image", meta.LayoutImage, "/room/image/SC.1102.jpg"),
		newField("Google Maps link", meta.MapURL, "https://maps.app.goo.gl/..."),
		newField("Reference images, comma-separated", strings.Join(meta.Images, ", "), "/room/image/a.jpg, /room/image/b.jpg"),
		newField("Title (optional)", meta.Title, ""),
		newField("Description (optional)", meta.Description, ""),
		newField("Latitude, longitude (optional)", latLng, "16.4747, 102.8237"),
	}
	title := "Room " + id
	if id == "" {
		title = "Add a room"
		fields = append([]field{newField("Room ID", "", "SC.1102")}, fields...)
	}
	return newForm(title, func(v []string) (tea.Cmd, error) {
		roomID := id
		if id == "" {
			roomID, v = v[0], v[1:]
			if roomID == "" {
				return nil, errors.New("the room ID is required")
			}
			if _, exists := m.roomMap[roomID]; exists {
				return nil, fmt.Errorf("room %s already exists", roomID)
			}
		}
		next := room.Meta{
			LayoutFile:  v[0],
			LayoutImage: v[1],
			MapURL:      v[2],
			Images:      splitList(v[3]),
			Title:       v[4],
			Description: v[5],
		}
		if next.LayoutFile == "" {
			next.LayoutFile = nonAlnum.ReplaceAllString(roomID, "") + ".json"
		}
		if v[6] != "" {
			lat, lng, err := parseLatLng(v[6])
			if err != nil {
				return nil, err
			}
			next.Lat, next.Lng = &lat, &lng
		}
		return change("Saved room "+roomID, false, true, func() error { return m.rooms.SaveRoom(roomID, next) }), nil
	}, fields...)
}

func splitList(s string) []string {
	out := []string{}
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseLatLng(s string) (float64, float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 == nil && err2 == nil {
			return lat, lng, nil
		}
	}
	return 0, 0, errors.New(`latitude, longitude must look like "16.4747, 102.8237"`)
}

// startLayoutEdit opens a pretty-printed copy of the room's layout in $EDITOR.
func (m Model) startLayoutEdit(id string) (tea.Model, tea.Cmd) {
	file := m.roomMap[id].LayoutFile
	if file == "" {
		m.setStatus("", fmt.Errorf("room %s has no layout file; set one with enter first", id))
		return m, nil
	}
	data, err := m.rooms.Layout(file)
	if err != nil {
		m.setStatus("", err)
		return m, nil
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		data = append(pretty.Bytes(), '\n')
	}
	tmp, err := os.CreateTemp("", "layout-*.json")
	if err == nil {
		_, err = tmp.Write(data)
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		m.setStatus("", err)
		return m, nil
	}
	return m, m.openEditor(layoutEdit{roomID: id, file: file, tmp: tmp.Name(), original: data})
}

func (m Model) openEditor(edit layoutEdit) tea.Cmd {
	return tea.ExecProcess(m.editor(edit.tmp), func(err error) tea.Msg { return editorMsg{edit: edit, err: err} })
}

// finishLayoutEdit saves the edited copy, or offers to fix it when invalid.
func (m Model) finishLayoutEdit(msg editorMsg) (tea.Model, tea.Cmd) {
	edit := msg.edit
	if msg.err != nil {
		os.Remove(edit.tmp)
		m.setStatus("", fmt.Errorf("editor failed: %w", msg.err))
		return m, nil
	}
	body, err := os.ReadFile(edit.tmp)
	if err != nil {
		m.setStatus("", err)
		return m, nil
	}
	if bytes.Equal(body, edit.original) {
		os.Remove(edit.tmp)
		m.setStatus("No changes to "+edit.file, nil)
		return m, nil
	}
	_, err = m.rooms.SaveLayout(edit.file, body)
	if errors.Is(err, roomconfig.ErrInvalidLayout) {
		edit.problem = err.Error()
		m.editing, m.mode = &edit, modeLayoutInvalid
		return m, nil
	}
	os.Remove(edit.tmp)
	if err == nil {
		m.setStatus("Saved layout "+edit.file+liveSoon, nil)
	} else {
		m.setStatus("", err)
	}
	return m, nil
}

func (m Model) updateLayoutInvalid(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	edit := *m.editing
	switch key.String() {
	case "e", "enter":
		m.editing, m.mode = nil, modeList
		return m, m.openEditor(edit)
	case "x", "esc":
		os.Remove(edit.tmp)
		m.editing, m.mode = nil, modeList
		m.setStatus("Discarded the changes to "+edit.file, nil)
	}
	return m, nil
}

func (m Model) layoutInvalidView() string {
	return errorStyle.Render("The layout was not saved: "+m.editing.problem) + "\n\n" +
		helpStyle.Render("e edit again · x discard the changes")
}

func (m Model) roomsView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Rooms (%d)", len(m.roomMap))) + "\n")
	if m.searching || m.search.Value() != "" {
		b.WriteString(m.search.View() + "\n")
	}
	b.WriteString("\n")
	if len(m.roomIDs) == 0 {
		b.WriteString(dimStyle.Render("No rooms. Press n to add one.") + "\n")
	}
	for i, id := range m.roomIDs {
		meta := m.roomMap[id]
		line := fmt.Sprintf("%-16s %-22s %s", id, meta.LayoutFile, truncate(meta.Title, 30))
		b.WriteString(cursorLine(i == m.roomCursor, line) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("↑↓ select · enter details · e edit layout in $EDITOR · p preview · n add · d delete · / filter · u reload site · ←→ tabs · q quit"))
	b.WriteString("\n" + dimStyle.Render("Drag-and-drop layout editing: run `operator web`."))
	return b.String()
}
