package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// field is one labelled input of a form.
type field struct {
	label string
	input textinput.Model
	// path fields suggest files and directories as you type (tab accepts).
	path bool
}

func newField(label, value, placeholder string) field {
	in := textinput.New()
	in.SetValue(value)
	in.Placeholder = placeholder
	in.CharLimit = 512
	in.Width = 60
	return field{label: label, input: in}
}

func newPathField(label, value string) field {
	f := newField(label, value, "path to .xlsx, .xls, .pdf or .json")
	f.path = true
	f.input.ShowSuggestions = true
	f.input.SetSuggestions(pathSuggestions(value))
	return f
}

// form is a list of fields edited one at a time; enter submits, esc cancels.
type form struct {
	title  string
	fields []field
	focus  int
	err    string
	// submit validates and runs the form; a non-nil error stays on screen.
	submit func(values []string) (tea.Cmd, error)
}

func newForm(title string, submit func([]string) (tea.Cmd, error), fields ...field) *form {
	f := &form{title: title, fields: fields, submit: submit}
	f.fields[0].input.Focus()
	return f
}

func (f *form) values() []string {
	out := make([]string, len(f.fields))
	for i, fl := range f.fields {
		out[i] = strings.TrimSpace(fl.input.Value())
	}
	return out
}

func (f *form) move(delta int) {
	f.fields[f.focus].input.Blur()
	f.focus = (f.focus + delta + len(f.fields)) % len(f.fields)
	f.fields[f.focus].input.Focus()
}

// update handles a message; done reports that the form closed (submitted or
// cancelled).
func (f *form) update(msg tea.Msg) (cmd tea.Cmd, done bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		cur := &f.fields[f.focus]
		switch key.String() {
		case "esc":
			return nil, true
		case "enter":
			cmd, err := f.submit(f.values())
			if err != nil {
				f.err = err.Error()
				return nil, false
			}
			return cmd, true
		case "tab":
			if cur.path && len(cur.input.MatchedSuggestions()) > 0 && cur.input.CurrentSuggestion() != cur.input.Value() {
				break // accept the suggestion below
			}
			f.move(1)
			return nil, false
		case "shift+tab", "up":
			f.move(-1)
			return nil, false
		case "down":
			f.move(1)
			return nil, false
		}
	}
	cur := &f.fields[f.focus]
	before := cur.input.Value()
	var c tea.Cmd
	cur.input, c = cur.input.Update(msg)
	if cur.path && cur.input.Value() != before {
		cur.input.SetSuggestions(pathSuggestions(cur.input.Value()))
	}
	return c, false
}

func (f *form) view() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(f.title) + "\n\n")
	for i, fl := range f.fields {
		label := fl.label
		if i == f.focus {
			label = selectedStyle.Render("› " + label)
		} else {
			label = "  " + label
		}
		b.WriteString(label + "\n  " + fl.input.View() + "\n")
	}
	if f.err != "" {
		b.WriteString("\n" + errorStyle.Render(f.err) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("tab/↑↓ move · tab accepts a path suggestion · enter save · esc cancel"))
	return b.String()
}

// pathSuggestions lists the entries of the directory being typed whose names
// start with the typed prefix; directories end in "/".
func pathSuggestions(value string) []string {
	dir, prefix := filepath.Split(value)
	listDir := dir
	if listDir == "" {
		listDir = "."
	}
	entries, err := os.ReadDir(listDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			name += "/"
		}
		out = append(out, dir+name)
	}
	sort.Strings(out)
	return out
}
