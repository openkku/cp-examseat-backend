package tui

import "testing"

func TestRenderPreviewNumbersLikeSeatMap(t *testing.T) {
	layout := `{
		"frontLabel": "Board",
		"layout": [
			{"type": "column", "label": "C1", "items": [
				{"type": "seats", "char": "A", "count": 3},
				{"type": "gap", "count": 1},
				{"type": "seats", "char": "B", "start": 10, "count": 2, "inverse": true}
			]},
			{"type": "aisle", "width": 20},
			{"type": "column", "label": "C2", "items": [
				{"type": "seats", "char": "A", "count": 2},
				{"type": "obstruction", "label": "Pillar"},
				{"type": "seats", "char": "C", "manual": [5, 7], "inverse": true}
			]}
		]
	}`
	got, err := renderPreview([]byte(layout))
	if err != nil {
		t.Fatal(err)
	}
	// A continues from 4 in the second column; B counts down; C is manual, reversed.
	want := "" +
		"   ▲ Board\n" +
		"C1      C2\n" +
		"A1      A4\n" +
		"A2      A5\n" +
		"A3      Pilla\n" +
		"        C7\n" +
		"B10     C5\n" +
		"B9\n" +
		"    ▼ Back"
	if got != want {
		t.Errorf("preview =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderPreviewRejectsInvalidJSON(t *testing.T) {
	if _, err := renderPreview([]byte(`{"layout": [`)); err == nil {
		t.Error("expected an error")
	}
}
