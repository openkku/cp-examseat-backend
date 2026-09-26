package xlsx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"

	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

// RawPythonEntry matches the JSON output structure of xslx_scrapler.py
type RawPythonEntry struct {
	Sheet     string  `json:"sheet"`
	Time      *string `json:"time"`
	Room      *string `json:"room"`
	Subject   *string `json:"subject"`
	Section   *string `json:"section"`
	StudentID string  `json:"student_id"`
	Seat      string  `json:"seat"`
	Note      *string `json:"note"`
}

func TestExtractorEquivalence(t *testing.T) {
	projectRoot := "../../.."
	xlsxPrefix := "final_2_2568"
	sourceDir := filepath.Join(projectRoot, "data", "source")

	xlsxFilename := filepath.Join(sourceDir, xlsxPrefix+".xlsx")
	if _, err := os.Stat(xlsxFilename); os.IsNotExist(err) {
		t.Skipf("Skipping equivalence test: %s not found", xlsxFilename)
	}

	runEquivalenceTest(t, xlsxPrefix, sourceDir)
}

func TestExtractorEquivalenceMock(t *testing.T) {
	projectRoot := "../../.."
	xlsxPrefix := "test_mock"
	sourceDir := filepath.Join(projectRoot, "data")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	xlsxFilename := filepath.Join(sourceDir, xlsxPrefix+".xlsx")
	createMockExcelFile(t, xlsxFilename)
	defer func() {
		t.Logf("Cleaning up mock Excel file: %s", xlsxFilename)
		os.Remove(xlsxFilename)
	}()

	runEquivalenceTest(t, xlsxPrefix, sourceDir)
}

func runEquivalenceTest(t *testing.T, xlsxPrefix string, sourceDir string) {
	projectRoot := "../../.."
	xlsxFilename := filepath.Join(sourceDir, xlsxPrefix+".xlsx")

	// Resolve absolute path of Python script to prevent directory navigation mismatch
	absScriptPath, err := filepath.Abs(filepath.Join(projectRoot, "script", "xslx_scrapler.py"))
	if err != nil {
		t.Fatalf("Failed to resolve script path: %v", err)
	}

	// 2. Run Python Scraper script inside the sourceDir directory (which is ignored by Git)
	// so that no temporary files pollution happens in the project root.
	t.Log("Running Python scraper...")
	cmd := exec.Command("python3", absScriptPath, xlsxPrefix)
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Python scraper failed: %v", err)
	}

	// Cleanup generated Python files inside sourceDir at test end
	pythonDataDir := filepath.Join(sourceDir, xlsxPrefix+"_data")
	defer func() {
		t.Logf("Cleaning up Python generated directory: %s", pythonDataDir)
		os.RemoveAll(pythonDataDir)
	}()

	// 3. Read generated Python JSON output
	pythonJSONPath := filepath.Join(pythonDataDir, xlsxPrefix+".json")
	jsonBytes, err := os.ReadFile(pythonJSONPath)
	if err != nil {
		t.Fatalf("Failed to read Python JSON output: %v", err)
	}

	var pythonEntries []RawPythonEntry
	if err := json.Unmarshal(jsonBytes, &pythonEntries); err != nil {
		t.Fatalf("Failed to unmarshal Python JSON: %v", err)
	}

	// 4. Open temporary SQLite database for Go Ingestion
	testDBPath := filepath.Join(t.TempDir(), "test_exams.db")
	db, err := sqlite.New(testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	// 5. Run Go direct Excel Ingestion
	t.Log("Running Go direct Excel extractor...")
	roundID := "test_round"
	err = ExtractAndMigrate(db, xlsxFilename, roundID, "Test Round Label")
	if err != nil {
		t.Fatalf("Go direct extractor failed: %v", err)
	}

	// 6. Query ingested exams from SQLite DB using a verification connection
	queryDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite for verification: %v", err)
	}
	defer queryDB.Close()

	rows, err := queryDB.Query(`
		SELECT 
			es.sheet, es.date, st_seat.student_id, (es.time_start || '-' || COALESCE(es.time_end, '')), es.room, es.subject_id, 
			COALESCE(s.name, '') as subject_name, 
			es.section, st_seat.seat, es.note 
		FROM exam_seats st_seat
		JOIN exam_sessions es ON st_seat.session_id = es.id
		JOIN students st ON st_seat.student_id = st.id
		LEFT JOIN subjects s ON es.subject_id = s.id AND es.exam_round = s.exam_round
		WHERE es.exam_round = ?`, roundID)
	if err != nil {
		t.Fatalf("Failed to query exams: %v", err)
	}
	defer rows.Close()

	type DBEntry struct {
		Sheet     string
		StudentID string
		Time      string
		Room      string
		Subject   string
		Section   string
		Seat      string
		Note      string
	}

	dbEntries := make(map[string]DBEntry)
	for rows.Next() {
		var e DBEntry
		var dummySubjName string
		var date string
		err := rows.Scan(&e.Sheet, &date, &e.StudentID, &e.Time, &e.Room, &e.Subject, &dummySubjName, &e.Section, &e.Seat, &e.Note)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		// Index by sheet, studentID, and subject since they form the unique key
		key := fmt.Sprintf("%s:%s:%s", e.Sheet, e.StudentID, e.Subject)
		dbEntries[key] = e
	}

	// 8. Assert equivalence between Python and Go outputs
	mismatches := 0
	for _, py := range pythonEntries {
		cleanStudentID := models.NormalizeStudentID(py.StudentID)
		if cleanStudentID == "" {
			continue // Old migrator also skips empty student IDs
		}

		pySubject := ""
		if py.Subject != nil {
			pySubject = strings.TrimSpace(*py.Subject)
		}

		key := fmt.Sprintf("%s:%s:%s", py.Sheet, cleanStudentID, pySubject)
		goEntry, exists := dbEntries[key]
		if !exists {
			t.Errorf("Mismatch: Exam in Python JSON (Key: %s) was not found in Go SQLite database", key)
			mismatches++
			if mismatches > 10 {
				t.Fatalf("Too many mismatches. Aborting test.")
			}
			continue
		}

		// Verify fields
		pyTime := ""
		if py.Time != nil {
			pyTime = strings.TrimSpace(*py.Time)
		}
		pyRoom := ""
		if py.Room != nil {
			pyRoom = strings.TrimSpace(*py.Room)
		}
		pySection := ""
		if py.Section != nil {
			pySection = strings.TrimSpace(strings.ReplaceAll(*py.Section, "SEC.", ""))
		}
		pySeat := strings.TrimSpace(py.Seat)
		pyNote := ""
		if py.Note != nil {
			pyNote = strings.TrimSpace(*py.Note)
		}

		if goEntry.Time != pyTime {
			t.Errorf("[%s] Time mismatch: Go %q vs Python %q", key, goEntry.Time, pyTime)
			mismatches++
		}
		if goEntry.Room != pyRoom {
			t.Errorf("[%s] Room mismatch: Go %q vs Python %q", key, goEntry.Room, pyRoom)
			mismatches++
		}
		if goEntry.Section != pySection {
			t.Errorf("[%s] Section mismatch: Go %q vs Python %q", key, goEntry.Section, pySection)
			mismatches++
		}
		if goEntry.Seat != pySeat {
			t.Errorf("[%s] Seat mismatch: Go %q vs Python %q", key, goEntry.Seat, pySeat)
			mismatches++
		}
		if goEntry.Note != pyNote {
			t.Errorf("[%s] Note mismatch: Go %q vs Python %q", key, goEntry.Note, pyNote)
			mismatches++
		}

		if mismatches > 10 {
			t.Fatalf("Too many mismatches. Aborting test.")
		}
	}

	if mismatches == 0 {
		t.Logf("✅ Verified all %d parsed records successfully! Exact equivalence found between Python and Go pipelines.", len(pythonEntries))
	} else {
		t.Fail()
	}
}

func TestExtractAndMigrateSuccess(t *testing.T) {
	projectRoot := "../../.."
	dataDir := filepath.Join(projectRoot, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Create a SQLite database in the ignored data folder
	testDBPath := filepath.Join(dataDir, "test_exams_mock.db")
	db, err := sqlite.New(testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	tempExcelPath := filepath.Join(dataDir, "test_mock.xlsx")
	t.Logf("Mock Excel file location: %s", tempExcelPath)
	t.Logf("Mock SQLite database location: %s", testDBPath)
	createMockExcelFile(t, tempExcelPath)

	// Run migration
	roundID := "mock_round"
	err = ExtractAndMigrate(db, tempExcelPath, roundID, "Mock Round Label")
	if err != nil {
		t.Fatalf("ExtractAndMigrate failed: %v", err)
	}

	// Query from db to verify
	queryDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite for verification: %v", err)
	}
	defer queryDB.Close()

	rows, err := queryDB.Query(`
		SELECT 
			es.sheet, es.date, st_seat.student_id, (es.time_start || '-' || COALESCE(es.time_end, '')), es.room, es.subject_id, 
			COALESCE(s.name, '') as subject_name, 
			es.section, st_seat.seat, es.note,
			COALESCE(st.branch, '') as branch
		FROM exam_seats st_seat
		JOIN exam_sessions es ON st_seat.session_id = es.id
		JOIN students st ON st_seat.student_id = st.id
		LEFT JOIN subjects s ON es.subject_id = s.id AND es.exam_round = s.exam_round
		WHERE es.exam_round = ?
		ORDER BY es.sheet ASC, es.date ASC, st_seat.student_id ASC`, roundID)
	if err != nil {
		t.Fatalf("Failed to query exams: %v", err)
	}
	defer rows.Close()

	type SeatRecord struct {
		Sheet       string
		Date        string
		StudentID   string
		Time        string
		Room        string
		Subject     string
		SubjectName string
		Section     string
		Seat        string
		Note        string
		Branch      string
	}

	var records []SeatRecord
	for rows.Next() {
		var r SeatRecord
		err := rows.Scan(&r.Sheet, &r.Date, &r.StudentID, &r.Time, &r.Room, &r.Subject, &r.SubjectName, &r.Section, &r.Seat, &r.Note, &r.Branch)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		records = append(records, r)
	}

	// Total records should be 4 (2 in sheet1 block1, 1 in sheet1 block2, 1 in sheet2)
	if len(records) != 4 {
		t.Errorf("Expected 4 records, got %d", len(records))
		for i, rec := range records {
			t.Logf("Record %d: %+v", i, rec)
		}
	}

	// Check Record 1: Sheet 2, Student 6533801234
	r1 := records[0]
	if r1.Sheet != "1 เม.ย 69 (เช้า)" || r1.Date != "2026-04-01" || r1.StudentID != "6533801234" || r1.Subject != "MATH101" || r1.SubjectName != "Calculus" || r1.Section != "3" || r1.Seat != "C12" || r1.Room != "SC.2102" || r1.Time != "09.00-12.00" || r1.Branch != "CP-AI" {
		t.Errorf("r1 mismatch: %+v", r1)
	}

	// Check Record 2: Sheet 1, Student 6833800746
	r2 := records[1]
	if r2.Sheet != "27 ต.ค68 (เช้า)" || r2.Date != "2025-10-27" || r2.StudentID != "6833800746" || r2.Subject != "CP9127" || r2.SubjectName != "Object Oriented Programming" || r2.Section != "18" || r2.Seat != "A4" || r2.Room != "CP.9127" || r2.Time != "08.30-11.30" || r2.Branch != "CP-CS" {
		t.Errorf("r2 mismatch: %+v", r2)
	}

	// Check Record 3: Sheet 1, Student 6833800770
	r3 := records[2]
	if r3.Sheet != "27 ต.ค68 (เช้า)" || r3.Date != "2025-10-27" || r3.StudentID != "6833800770" || r3.Note != "หมดสิทธิ์สอบ" || r3.Seat != "A5" || r3.Branch != "CP-Cy" {
		t.Errorf("r3 mismatch: %+v", r3)
	}

	// Check Record 4: Sheet 1, Student 6833803215
	r4 := records[3]
	if r4.Sheet != "27 ต.ค68 (เช้า)" || r4.Date != "2025-10-27" || r4.StudentID != "6833803215" || r4.Subject != "LI101002" || r4.SubjectName != "English II" || r4.Section != "90" || r4.Seat != "B1" || r4.Room != "SC.1101" || r4.Time != "13.00-16.00" || r4.Branch != "LI-CS" {
		t.Errorf("r4 mismatch: %+v", r4)
	}
}

func TestExtractAndMigrateSkipHiddenRows(t *testing.T) {
	projectRoot := "../../.."
	dataDir := filepath.Join(projectRoot, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	testDBPath := filepath.Join(dataDir, "test_exams_hidden.db")
	db, err := sqlite.New(testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	tempExcelPath := filepath.Join(dataDir, "test_hidden_mock.xlsx")
	defer os.Remove(tempExcelPath)

	f := excelize.NewFile()
	sheet := "24 ส.ค. 69 (เช้า)"
	f.NewSheet(sheet)

	// Block 1 (Hidden rows 1-5)
	f.SetCellValue(sheet, "A1", "ใบรายชื่อ")
	f.SetCellValue(sheet, "A2", "รายวิชา")
	f.SetCellValue(sheet, "B2", "HIDDEN101 Hidden Subject")
	f.SetCellValue(sheet, "B3", "รหัส")
	f.SetCellValue(sheet, "A4", "1")
	f.SetCellValue(sheet, "B4", "993380001-1")
	f.SetCellValue(sheet, "E4", "H1")
	for r := 1; r <= 5; r++ {
		f.SetRowVisible(sheet, r, false)
	}

	// Block 2 (Visible rows 10-14)
	f.SetCellValue(sheet, "A10", "ใบรายชื่อ")
	f.SetCellValue(sheet, "A11", "รายวิชา")
	f.SetCellValue(sheet, "B11", "VIS101 Visible Subject")
	f.SetCellValue(sheet, "B12", "รหัส")
	f.SetCellValue(sheet, "A13", "1")
	f.SetCellValue(sheet, "B13", "883380002-2")
	f.SetCellValue(sheet, "E13", "V1")

	f.DeleteSheet("Sheet1")
	if err := f.SaveAs(tempExcelPath); err != nil {
		t.Fatalf("Failed to save hidden mock Excel file: %v", err)
	}

	err = ExtractAndMigrate(db, tempExcelPath, "hidden_test_round", "Hidden Test")
	if err != nil {
		t.Fatalf("ExtractAndMigrate failed: %v", err)
	}

	queryDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer queryDB.Close()

	rows, err := queryDB.Query(`SELECT st_seat.student_id, st_seat.seat, es.subject_id FROM exam_seats st_seat JOIN exam_sessions es ON st_seat.session_id = es.id WHERE es.exam_round = 'hidden_test_round'`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var stID, stSeat, stSubj string
		if err := rows.Scan(&stID, &stSeat, &stSubj); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		count++
		if stID == "9933800011" {
			t.Errorf("Found hidden student 9933800011 which should have been skipped!")
		}
		if stID != "8833800022" || stSeat != "V1" || stSubj != "VIS101" {
			t.Errorf("Unexpected student record: id=%s, seat=%s, subj=%s", stID, stSeat, stSubj)
		}
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 visible seat record, got %d", count)
	}
}

func createMockExcelFile(t *testing.T, filePath string) {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: "27 ต.ค68 (เช้า)" (no dot, no space) -> 2025-10-27
	sheet1 := "27 ต.ค68 (เช้า)"
	f.NewSheet(sheet1)

	// Block 1 on Sheet 1
	f.SetCellValue(sheet1, "A1", "ใบรายชื่อนักศึกษาสอบกลางภาค")
	f.SetCellValue(sheet1, "A2", "รายวิชา")
	f.SetCellValue(sheet1, "B2", "CP9127\u00a0Object Oriented Programming")
	f.SetCellValue(sheet1, "F2", "SEC.18")
	f.SetCellValue(sheet1, "A3", "เวลาสอบ")
	f.SetCellValue(sheet1, "B3", "08.30-11.30 น.")
	f.SetCellValue(sheet1, "A4", "ห้องสอบ")
	f.SetCellValue(sheet1, "B4", "CP.9127")
	f.SetCellValue(sheet1, "B5", "รหัส")

	// Students for Block 1
	f.SetCellValue(sheet1, "A6", "1")
	f.SetCellValue(sheet1, "B6", "683380074-6")
	f.SetCellValue(sheet1, "D6", "CP-CS")
	f.SetCellValue(sheet1, "E6", "A4")
	f.SetCellValue(sheet1, "F6", "")

	f.SetCellValue(sheet1, "A7", "2")
	f.SetCellValue(sheet1, "B7", "683380077-0")
	f.SetCellValue(sheet1, "D7", "CP-Cy")
	f.SetCellValue(sheet1, "E7", "A5")
	f.SetCellValue(sheet1, "F7", "หมดสิทธิ์สอบ")

	// Empty rows to separate blocks
	f.SetCellValue(sheet1, "B8", "")
	f.SetCellValue(sheet1, "E8", "")

	// Block 2 on Sheet 1 (starts at row 10)
	f.SetCellValue(sheet1, "A10", "ใบรายชื่อนักศึกษา")
	f.SetCellValue(sheet1, "A11", "รายวิชา")
	f.SetCellValue(sheet1, "B11", "LI101002 English II") // normal space
	f.SetCellValue(sheet1, "F11", "SEC. 90")             // space in section
	f.SetCellValue(sheet1, "A12", "เวลาสอบ")
	f.SetCellValue(sheet1, "B12", "13.00-16.00 น.")
	f.SetCellValue(sheet1, "A13", "ห้องสอบ")
	f.SetCellValue(sheet1, "B13", "SC.1101")
	f.SetCellValue(sheet1, "B14", "รหัส")

	// Students for Block 2
	f.SetCellValue(sheet1, "A15", "1")
	f.SetCellValue(sheet1, "B15", "683380321-5")
	f.SetCellValue(sheet1, "D15", "LI-CS")
	f.SetCellValue(sheet1, "E15", "B1")
	f.SetCellValue(sheet1, "F15", "")

	// Empty row to terminate block 2
	f.SetCellValue(sheet1, "B16", "")
	f.SetCellValue(sheet1, "E16", "")

	// Sheet 2: "1 เม.ย 69 (เช้า)" (no dot at end, space) -> 2026-04-01
	sheet2 := "1 เม.ย 69 (เช้า)"
	f.NewSheet(sheet2)

	f.SetCellValue(sheet2, "A1", "ใบรายชื่อ")
	f.SetCellValue(sheet2, "A2", "รายวิชา")
	f.SetCellValue(sheet2, "B2", "MATH101 Calculus")
	f.SetCellValue(sheet2, "F2", "SEC.3")
	f.SetCellValue(sheet2, "A3", "เวลาสอบ")
	f.SetCellValue(sheet2, "B3", "09.00-12.00 น.")
	f.SetCellValue(sheet2, "A4", "ห้องสอบ")
	f.SetCellValue(sheet2, "B4", "SC.2102")
	f.SetCellValue(sheet2, "B5", "รหัส")
	f.SetCellValue(sheet2, "A6", "1")
	f.SetCellValue(sheet2, "B6", "653380123-4")
	f.SetCellValue(sheet2, "D6", "CP-AI")
	f.SetCellValue(sheet2, "E6", "C12")

	// Delete default sheet
	f.DeleteSheet("Sheet1")

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save mock Excel file: %v", err)
	}
}
