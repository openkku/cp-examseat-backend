package tui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

type fakeRounds struct {
	list     []admin.RoundSummary
	renamed  map[string]string
	deleted  []string
	imported []admin.ImportRequest
	reloads  int
}

func (f *fakeRounds) Rounds(context.Context) ([]admin.RoundSummary, error) { return f.list, nil }
func (f *fakeRounds) RenameRound(_ context.Context, id, label string) (string, error) {
	f.renamed[id] = label
	return label, nil
}
func (f *fakeRounds) DeleteRound(_ context.Context, id, customID string) error {
	f.deleted = append(f.deleted, id+"/"+customID)
	return nil
}
func (f *fakeRounds) ValidateFileType(ext string) error {
	if ext != ".json" {
		return errors.New("Unsupported file type (use .xlsx, .xls, .pdf or .json)")
	}
	return nil
}
func (f *fakeRounds) Import(_ context.Context, req admin.ImportRequest) (int, error) {
	f.imported = append(f.imported, req)
	return 3, nil
}
func (f *fakeRounds) Reload(context.Context) error { f.reloads++; return nil }

type fakeRooms struct {
	meta    map[string]room.Meta
	layouts map[string]string
	saved   map[string]room.Meta
}

func (f *fakeRooms) Rooms() (map[string]room.Meta, error) { return f.meta, nil }
func (f *fakeRooms) SaveRoom(id string, meta room.Meta) error {
	f.saved[id] = meta
	return nil
}
func (f *fakeRooms) DeleteRoom(id string) error         { delete(f.meta, id); return nil }
func (f *fakeRooms) Layout(file string) ([]byte, error) { return []byte(f.layouts[file]), nil }
func (f *fakeRooms) SaveLayout(file string, body []byte) (int, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		return 0, roomconfig.ErrInvalidLayout
	}
	f.layouts[file] = string(body)
	return 0, nil
}

func newTestModel(t *testing.T) (Model, *fakeRounds, *fakeRooms) {
	t.Helper()
	rounds := &fakeRounds{
		list: []admin.RoundSummary{
			{ID: "mid_1_2569", Label: "กลางภาค 1/2569", Seats: 10, CustomDatasets: []admin.CustomDataset{{ID: "LAB_X", Seats: 2}}},
			{ID: "final_1_2569", Label: "ปลายภาค 1/2569", Seats: 5},
		},
		renamed: map[string]string{},
	}
	rooms := &fakeRooms{
		meta:    map[string]room.Meta{"CP.9127": {LayoutFile: "CP9127.json"}},
		layouts: map[string]string{"CP9127.json": `{"layout":[]}`},
		saved:   map[string]room.Meta{},
	}
	m := New(rounds, rooms, t.TempDir())
	m = run(t, m, m.Init())
	return m, rounds, rooms
}

// run executes cmd and feeds the app's own messages back into the model.
// Bubble Tea's internal messages (cursor blinks, external processes) are
// dropped: a blink re-arms itself forever, and tests send editorMsg directly.
func run(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-result:
	case <-time.After(100 * time.Millisecond):
		return m // a timer (cursor blink)
	}
	switch msg := msg.(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			m = run(t, m, c)
		}
	case roundsMsg, roomsMsg, doneMsg, editorMsg:
		next, c := m.Update(msg)
		m = run(t, next.(Model), c)
	}
	return m
}

// press sends keys and runs the resulting commands.
func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, cmd := m.Update(msg)
		m = run(t, next.(Model), cmd)
	}
	return m
}

func TestRenameRound(t *testing.T) {
	m, rounds, _ := newTestModel(t)
	m = press(t, m, "r")
	if m.mode != modeForm {
		t.Fatalf("mode = %v; want form", m.mode)
	}
	m.form.fields[0].input.SetValue("Midterm")
	m = press(t, m, "enter")
	if rounds.renamed["mid_1_2569"] != "Midterm" {
		t.Errorf("renamed = %v", rounds.renamed)
	}
	if m.mode != modeList || !strings.Contains(m.status, "Renamed mid_1_2569") || m.statusErr {
		t.Errorf("after rename: mode=%v status=%q", m.mode, m.status)
	}
}

func TestDeleteNeedsConfirmation(t *testing.T) {
	m, rounds, _ := newTestModel(t)
	m = press(t, m, "down", "d", "n")
	if len(rounds.deleted) != 0 {
		t.Fatalf("deleted without confirmation: %v", rounds.deleted)
	}
	m = press(t, m, "d", "y")
	if len(rounds.deleted) != 1 || rounds.deleted[0] != "final_1_2569/" {
		t.Errorf("deleted = %v", rounds.deleted)
	}

	// A custom dataset is deleted from the round's dataset view.
	m = press(t, m, "k", "enter", "d", "y")
	if len(rounds.deleted) != 2 || rounds.deleted[1] != "mid_1_2569/LAB_X" {
		t.Errorf("deleted = %v", rounds.deleted)
	}
	if m.mode != modeDatasets {
		t.Errorf("mode after dataset delete = %v; want datasets", m.mode)
	}
}

func TestImportFormValidation(t *testing.T) {
	m, rounds, _ := newTestModel(t)
	m = press(t, m, "i")

	set := func(values ...string) {
		for i, v := range values {
			m.form.fields[i].input.SetValue(v)
		}
	}

	set(m.sourceDir + "/")
	m = press(t, m, "enter")
	if m.mode != modeForm || m.form.err != "choose a file" {
		t.Fatalf("no file: mode=%v err=%q", m.mode, m.form.err)
	}
	set("/tmp/a.exe", "r")
	m = press(t, m, "enter")
	if !strings.Contains(m.form.err, "Unsupported file type") {
		t.Fatalf("bad type: err=%q", m.form.err)
	}
	set("/tmp/a.json", "")
	m = press(t, m, "enter")
	if m.form.err != "the round ID is required" {
		t.Fatalf("no round: err=%q", m.form.err)
	}

	set("/tmp/a.json", "final_2_2569", "ปลายภาค 2/2569", "LAB, Lab", "", "LAB_Y")
	m = press(t, m, "enter")
	if len(rounds.imported) != 1 {
		t.Fatalf("imported = %v", rounds.imported)
	}
	req := rounds.imported[0]
	if req.Round != "final_2_2569" || req.CustomID != "LAB_Y" || len(req.Labels) != 2 {
		t.Errorf("request = %+v", req)
	}
	if !strings.Contains(m.status, "Imported 3 seats into final_2_2569") {
		t.Errorf("status = %q", m.status)
	}
}

func TestImportSuggestsFiles(t *testing.T) {
	m, _, _ := newTestModel(t)
	if err := os.WriteFile(m.sourceDir+"/final.xlsx", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "i", "f", "i") // the path field starts in sourceDir
	m = press(t, m, "tab")
	if got := m.form.fields[0].input.Value(); got != m.sourceDir+"/final.xlsx" {
		t.Errorf("tab completion = %q", got)
	}
	m = press(t, m, "tab") // no suggestion left: tab moves to the next field
	if m.form.focus != 1 {
		t.Errorf("focus = %d; want 1", m.form.focus)
	}
}

func TestEditRoomMetadata(t *testing.T) {
	m, _, rooms := newTestModel(t)
	m = press(t, m, "2", "enter")
	m.form.fields[4].input.SetValue("Lecture hall")
	m.form.fields[6].input.SetValue("16.47, 102.82")
	m = press(t, m, "enter")
	got := rooms.saved["CP.9127"]
	if got.LayoutFile != "CP9127.json" || got.Title != "Lecture hall" || got.Lat == nil || *got.Lng != 102.82 {
		t.Errorf("saved = %+v", got)
	}

	m = press(t, m, "enter")
	m.form.fields[6].input.SetValue("north")
	m = press(t, m, "enter")
	if m.mode != modeForm || !strings.Contains(m.form.err, "latitude") {
		t.Errorf("bad lat/lng accepted: mode=%v err=%q", m.mode, m.form.err)
	}
}

func TestLayoutEditValidatesAndReopens(t *testing.T) {
	m, _, rooms := newTestModel(t)
	var opened []string
	m.editor = func(path string) *exec.Cmd { opened = append(opened, path); return exec.Command("true") }

	m = press(t, m, "2", "e")
	if len(opened) != 1 {
		t.Fatalf("editor not opened: %v", opened)
	}
	tmp := opened[0]
	edit := layoutEdit{roomID: "CP.9127", file: "CP9127.json", tmp: tmp}
	edit.original, _ = os.ReadFile(tmp)

	// The operator saves invalid JSON: nothing is written, they may fix it.
	os.WriteFile(tmp, []byte("[oops"), 0o600)
	next, _ := m.Update(editorMsg{edit: edit})
	m = next.(Model)
	if m.mode != modeLayoutInvalid || rooms.layouts["CP9127.json"] != `{"layout":[]}` {
		t.Fatalf("invalid layout: mode=%v layout=%s", m.mode, rooms.layouts["CP9127.json"])
	}
	m = press(t, m, "e")
	if len(opened) != 2 || opened[1] != tmp {
		t.Fatalf("re-edit should reopen the same copy: %v", opened)
	}

	os.WriteFile(tmp, []byte(`{"layout":[{"type":"aisle","width":20}]}`), 0o600)
	next, _ = m.Update(editorMsg{edit: edit})
	m = next.(Model)
	if !strings.Contains(rooms.layouts["CP9127.json"], "aisle") || !strings.Contains(m.status, "Saved layout CP9127.json") {
		t.Errorf("valid layout not saved: %s / %q", rooms.layouts["CP9127.json"], m.status)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temporary copy left behind")
	}
}

func TestPreviewAndReload(t *testing.T) {
	m, rounds, _ := newTestModel(t)
	m = press(t, m, "2", "p")
	if m.mode != modePreview || !strings.Contains(m.preview, "▲ Front") {
		t.Errorf("preview: mode=%v %q", m.mode, m.preview)
	}
	m = press(t, m, "x", "u")
	if rounds.reloads != 1 || !strings.Contains(m.status, "reload") {
		t.Errorf("reload: %d %q", rounds.reloads, m.status)
	}
}
