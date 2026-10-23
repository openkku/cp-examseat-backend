package operator_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

const importJSON = `[
 {"Sheet": "S9", "Date": "2026-10-01", "Time": "08.30-11.30", "Room": "CP.9127", "Subject": "CP9999", "SubjectName": "Imported", "Section": "1", "StudentID": "6733809999", "Seat": "A3"}
]`

func roundIDs(t *testing.T, o *testOperator) map[string]map[string]any {
	t.Helper()
	rec := o.json(http.MethodGet, "/api/rounds/", "")
	var rounds []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rounds); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("rounds = %d %s", rec.Code, rec.Body)
	}
	out := make(map[string]map[string]any, len(rounds))
	for _, r := range rounds {
		out[r["id"].(string)] = r
	}
	return out
}

func TestImportSignalsReload(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.upload(map[string]string{"round": "final_1_2569", "display": "ปลายภาค 1/2569"}, "seats.json", importJSON, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"seats":1`) {
		t.Fatalf("import = %d %s", rec.Code, rec.Body)
	}
	if !o.reloadSignalled() {
		t.Error("import did not signal the server to reload")
	}
	if r, ok := roundIDs(t, o)["final_1_2569"]; !ok || r["label"] != "ปลายภาค 1/2569" {
		t.Errorf("imported round missing: %v", r)
	}
}

func TestPreviewImportDoesNotSaveOrSignalReload(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.uploadTo("/api/rounds/import/preview", map[string]string{"round": "preview_round", "display": "Preview Test"}, "seats.json", importJSON, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview import = %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"seats":1`) || !strings.Contains(body, `"students":1`) || !strings.Contains(body, `"CP.9127"`) {
		t.Errorf("unexpected preview body: %s", body)
	}
	if o.reloadSignalled() {
		t.Error("preview import must not signal reload")
	}
	if _, ok := roundIDs(t, o)["preview_round"]; ok {
		t.Error("preview import must not save round into database")
	}
}

func TestPreviewRoutes(t *testing.T) {
	o := newOperator(t, 1<<20)

	// Test dates options
	rec := o.serve(request(http.MethodGet, "/api/preview/options?type=dates&round=mid_1_2569&room=CP.9127", "", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("preview options dates = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "2026-09-01") {
		t.Errorf("expected date 2026-09-01 in options, got %s", rec.Body)
	}

	// Test times options
	rec = o.serve(request(http.MethodGet, "/api/preview/options?type=times&round=mid_1_2569&room=CP.9127&date=2026-09-01", "", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("preview options times = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "08.30-11.30") {
		t.Errorf("expected time 08.30-11.30 in options, got %s", rec.Body)
	}

	// Test preview seats
	rec = o.serve(request(http.MethodGet, "/api/preview/seats?round=mid_1_2569&room=CP.9127&date=2026-09-01&time=08.30-11.30", "", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("preview seats = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"seat":"A1"`) || !strings.Contains(rec.Body.String(), `"student_id":"6533801234"`) {
		t.Errorf("expected seat A1 for student in roster, got %s", rec.Body)
	}

	// Missing params returns 400
	rec = o.serve(request(http.MethodGet, "/api/preview/seats?round=mid_1_2569", "", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing preview seat params, got %d", rec.Code)
	}
}

func TestHealthCheckRoute(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.serve(request(http.MethodGet, "/api/health", "", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health check = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "problems") {
		t.Errorf("expected problems in health response, got %s", rec.Body)
	}
}

func TestImportValidation(t *testing.T) {
	o := newOperator(t, 1<<10)

	for _, tc := range []struct {
		name     string
		fields   map[string]string
		filename string
		content  string
		want     int
	}{
		{"missing round", map[string]string{}, "a.json", importJSON, http.StatusBadRequest},
		{"unsafe round id", map[string]string{"round": "../../x"}, "a.json", importJSON, http.StatusBadRequest},
		{"unsafe custom id", map[string]string{"round": "r", "custom_id": "a b"}, "a.json", importJSON, http.StatusBadRequest},
		{"missing file", map[string]string{"round": "r"}, "", "", http.StatusBadRequest},
		{"unsupported type", map[string]string{"round": "r"}, "a.exe", "MZ", http.StatusBadRequest},
		{"unparsable file", map[string]string{"round": "r"}, "a.json", "{not json", http.StatusUnprocessableEntity},
		{"too large", map[string]string{"round": "r"}, "a.json", strings.Repeat(" ", 4<<10), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := o.upload(tc.fields, tc.filename, tc.content, nil)
			if rec.Code != tc.want {
				t.Errorf("= %d %s; want %d", rec.Code, rec.Body, tc.want)
			}
			if strings.Contains(rec.Body.String(), os.TempDir()) {
				t.Errorf("error leaks the temp path: %s", rec.Body)
			}
		})
	}
	if o.reloadSignalled() {
		t.Error("a failed import signalled a reload")
	}
}

func TestRenameAndDelete(t *testing.T) {
	o := newOperator(t, 1<<20)
	o.upload(map[string]string{"round": "mid_1_2569", "custom_id": "LAB_X"}, "lab.json", importJSON, nil)
	o.reloadSignalled()

	if rec := o.json(http.MethodPatch, "/api/rounds/mid_1_2569", `{"label":"Midterm"}`); rec.Code != http.StatusOK {
		t.Fatalf("rename = %d %s", rec.Code, rec.Body)
	}
	if !o.reloadSignalled() {
		t.Error("rename did not signal a reload")
	}
	if got := roundIDs(t, o)["mid_1_2569"]["label"]; got != "Midterm" {
		t.Errorf("label = %v", got)
	}
	if rec := o.json(http.MethodPatch, "/api/rounds/nope", `{"label":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("rename unknown = %d", rec.Code)
	}

	// Deleting a custom dataset keeps the rest of the round.
	if rec := o.json(http.MethodDelete, "/api/rounds/mid_1_2569?custom_id=LAB_X", ""); rec.Code != http.StatusOK {
		t.Fatalf("delete dataset = %d %s", rec.Code, rec.Body)
	}
	r := roundIDs(t, o)["mid_1_2569"]
	if r == nil || r["seats"].(float64) != 1 || len(r["custom_datasets"].([]any)) != 0 {
		t.Errorf("after deleting the dataset: %v", r)
	}

	if rec := o.json(http.MethodDelete, "/api/rounds/mid_1_2569", ""); rec.Code != http.StatusOK {
		t.Fatalf("delete round = %d", rec.Code)
	}
	if len(roundIDs(t, o)) != 0 {
		t.Errorf("round still listed after delete")
	}
	if rec := o.json(http.MethodDelete, "/api/rounds/mid_1_2569", ""); rec.Code != http.StatusNotFound {
		t.Errorf("delete missing round = %d", rec.Code)
	}
	if rec := o.json(http.MethodDelete, "/api/rounds/bad%20id", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("delete bad id = %d", rec.Code)
	}
}

func TestRoomEditsSignalReload(t *testing.T) {
	o := newOperator(t, 1<<20)

	if rec := o.json(http.MethodPost, "/api/rooms/NEW.1", `{"layout_file":"new.json","layout_image":"","map_url":"","images":[]}`); rec.Code != http.StatusOK {
		t.Fatalf("save room = %d %s", rec.Code, rec.Body)
	}
	if !o.reloadSignalled() {
		t.Error("saving a room did not signal a reload")
	}
	if rec := o.json(http.MethodPost, "/api/rooms/layouts/new.json", `{"layout":[{"type":"aisle","width":1}]}`); rec.Code != http.StatusOK {
		t.Fatalf("save layout = %d %s", rec.Code, rec.Body)
	}
	if !o.reloadSignalled() {
		t.Error("saving a layout did not signal a reload")
	}
	if rec := o.json(http.MethodPost, "/api/rooms/layouts/new.json", `[1]`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid layout = %d", rec.Code)
	}
	if o.reloadSignalled() {
		t.Error("a rejected layout signalled a reload")
	}
	if rec := o.json(http.MethodPost, "/api/rounds/reload", ""); rec.Code != http.StatusOK || !o.reloadSignalled() {
		t.Errorf("manual reload = %d", rec.Code)
	}
}
