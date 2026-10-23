// Package tui is the operator app's terminal UI: round management and room
// editing over plain SSH, with no network port. It drives the same use cases
// as the web UI (admin.Service, roomconfig.Service).
package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/room"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	tabStyle      = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("245"))
	activeTab     = tabStyle.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("25")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

// liveSoon follows every successful change: the server reloads on its own.
const liveSoon = " — the site updates within a few seconds"

type tab int

const (
	tabRounds tab = iota
	tabRooms
)

// mode is what the active tab shows.
type mode int

const (
	modeList mode = iota
	modeDatasets
	modeForm
	modeConfirm
	modePreview
	modeLayoutInvalid
)

// Messages from asynchronous commands.
type (
	roundsMsg struct {
		rounds []admin.RoundSummary
		err    error
	}
	roomsMsg struct {
		rooms map[string]room.Meta
		err   error
	}
	// doneMsg reports a finished change and which lists to reload.
	doneMsg struct {
		text         string
		err          error
		reloadRounds bool
		reloadRooms  bool
	}
	// editorMsg reports that $EDITOR exited on a layout copy.
	editorMsg struct {
		edit layoutEdit
		err  error
	}
)

// layoutEdit is a layout file being edited through a temporary copy.
type layoutEdit struct {
	roomID, file, tmp string
	original          []byte
	problem           string
}

// Model is the root Bubble Tea model.
type Model struct {
	rounds    Rounds
	rooms     Rooms
	sourceDir string
	// editor builds the command that edits path ($EDITOR, default vi).
	editor func(path string) *exec.Cmd

	tab  tab
	mode mode
	// back is the mode a form or prompt returns to.
	back          mode
	width, height int
	status        string
	statusErr     bool

	roundList     []admin.RoundSummary
	roundCursor   int
	datasetCursor int

	roomMap    map[string]room.Meta
	roomIDs    []string
	roomCursor int
	search     textinput.Model
	searching  bool

	form    *form
	confirm *confirmation
	preview string
	editing *layoutEdit
}

// confirmation asks y/n before a destructive change.
type confirmation struct {
	prompt string
	yes    tea.Cmd
}

// New creates the TUI. sourceDir is where the import form starts browsing.
func New(rounds Rounds, rooms Rooms, sourceDir string) Model {
	search := textinput.New()
	search.Placeholder = "filter rooms"
	search.Prompt = "/ "
	return Model{
		rounds:    rounds,
		rooms:     rooms,
		sourceDir: sourceDir,
		editor:    defaultEditor,
		search:    search,
		roomMap:   map[string]room.Meta{},
	}
}

// Run shows the TUI on the terminal until the operator quits.
func Run(rounds Rounds, rooms Rooms, sourceDir string) error {
	_, err := tea.NewProgram(New(rounds, rooms, sourceDir), tea.WithAltScreen()).Run()
	return err
}

func defaultEditor(path string) *exec.Cmd {
	args := strings.Fields(os.Getenv("EDITOR"))
	if len(args) == 0 {
		args = []string{"vi"}
	}
	return exec.Command(args[0], append(args[1:], path)...) // #nosec G204 G702 -- the operator's own $EDITOR, run with their own rights
}

// Init loads both lists.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadRounds(), m.loadRooms())
}

func (m Model) loadRounds() tea.Cmd {
	return func() tea.Msg {
		r, err := m.rounds.Rounds(context.Background())
		return roundsMsg{r, err}
	}
}

func (m Model) loadRooms() tea.Cmd {
	return func() tea.Msg {
		r, err := m.rooms.Rooms()
		return roomsMsg{r, err}
	}
}

// change runs fn in the background and reports text on success.
func change(text string, reloadRounds, reloadRooms bool, fn func() error) tea.Cmd {
	return func() tea.Msg {
		return doneMsg{text: text, err: fn(), reloadRounds: reloadRounds, reloadRooms: reloadRooms}
	}
}

func (m *Model) setStatus(text string, err error) {
	if err != nil {
		m.status, m.statusErr = err.Error(), true
		return
	}
	m.status, m.statusErr = text, false
}

// Update handles messages for the whole app.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case roundsMsg:
		if msg.err != nil {
			m.setStatus("", msg.err)
		} else {
			m.roundList = msg.rounds
			m.roundCursor = clamp(m.roundCursor, len(m.roundList))
		}
		return m, nil
	case roomsMsg:
		if msg.err != nil {
			m.setStatus("", msg.err)
		} else {
			m.roomMap = msg.rooms
			m.refilter()
		}
		return m, nil
	case doneMsg:
		if msg.err == nil {
			msg.text += liveSoon
		}
		m.setStatus(msg.text, msg.err)
		var cmds []tea.Cmd
		if msg.reloadRounds {
			cmds = append(cmds, m.loadRounds())
		}
		if msg.reloadRooms {
			cmds = append(cmds, m.loadRooms())
		}
		return m, tea.Batch(cmds...)
	case editorMsg:
		return m.finishLayoutEdit(msg)
	}

	switch m.mode {
	case modeForm:
		cmd, done := m.form.update(msg)
		if done {
			m.form, m.mode = nil, m.back
		}
		return m, cmd
	case modeConfirm:
		if key, ok := msg.(tea.KeyMsg); ok {
			yes := m.confirm.yes
			m.confirm, m.mode = nil, m.back
			if key.String() == "y" || key.String() == "Y" {
				return m, yes
			}
			m.setStatus("Cancelled", nil)
		}
		return m, nil
	case modePreview:
		if _, ok := msg.(tea.KeyMsg); ok {
			m.preview, m.mode = "", modeList
		}
		return m, nil
	case modeLayoutInvalid:
		return m.updateLayoutInvalid(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.searching {
		return m.updateSearch(key)
	}
	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "1":
		m.tab, m.mode = tabRounds, modeList
		return m, nil
	case "2":
		m.tab, m.mode = tabRooms, modeList
		return m, nil
	case "left", "right", "shift+tab":
		if m.mode == modeList {
			m.tab = 1 - m.tab
		}
		return m, nil
	case "u":
		return m, change("Asked the server to reload", false, false, func() error { return m.rounds.Reload(context.Background()) })
	}
	if m.tab == tabRounds {
		return m.updateRounds(key)
	}
	return m.updateRooms(key)
}

func (m *Model) ask(prompt string, yes tea.Cmd) {
	m.back = m.mode
	m.confirm, m.mode = &confirmation{prompt: prompt, yes: yes}, modeConfirm
}

func (m *Model) openForm(f *form) {
	m.back = m.mode
	m.form, m.mode = f, modeForm
}

// View renders the app.
func (m Model) View() string {
	var b strings.Builder
	tabs := []string{"1 Rounds", "2 Rooms"}
	for i, t := range tabs {
		if tab(i) == m.tab {
			b.WriteString(activeTab.Render(t))
		} else {
			b.WriteString(tabStyle.Render(t))
		}
	}
	b.WriteString("  " + dimStyle.Render("Operator · CP Exam Seat") + "\n\n")

	switch m.mode {
	case modeForm:
		b.WriteString(m.form.view())
	case modeConfirm:
		b.WriteString(errorStyle.Render(m.confirm.prompt) + "\n\n" + helpStyle.Render("y confirm · any other key cancels"))
	case modePreview:
		b.WriteString(m.preview + "\n\n" + helpStyle.Render("any key closes"))
	case modeLayoutInvalid:
		b.WriteString(m.layoutInvalidView())
	default:
		if m.tab == tabRounds {
			b.WriteString(m.roundsView())
		} else {
			b.WriteString(m.roomsView())
		}
	}

	if m.status != "" {
		style := okStyle
		if m.statusErr {
			style = errorStyle
		}
		b.WriteString("\n\n" + style.Render(m.status))
	}
	return b.String()
}

func clamp(i, n int) int {
	if n == 0 || i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
