package operator_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	adminhandler "github.com/openkku/cp-examseat-backend/internal/admin/handler"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/datacheck"
	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer"
	"github.com/openkku/cp-examseat-backend/internal/operator"
	"github.com/openkku/cp-examseat-backend/internal/platform/reloadsignal"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
	roomfs "github.com/openkku/cp-examseat-backend/internal/roomconfig/filesystem"
	roomconfighandler "github.com/openkku/cp-examseat-backend/internal/roomconfig/handler"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// testOperator is the operator app wired on a fresh data directory with one
// seeded round, mid_1_2569.
type testOperator struct {
	t       *testing.T
	handler http.Handler
	dataDir string
	db      *sqlite.DB
}

func newOperator(t *testing.T, maxUpload int64) *testOperator {
	t.Helper()
	dataDir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dataDir, "exams.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	seats := []domain.Seat{{Sheet: "S1", Date: "2026-09-01", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP1001", Section: "1", StudentID: "6533801234", Seat: "A1"}}
	if err := db.AddRound(context.Background(), "mid_1_2569", "กลางภาค 1/2569", seats); err != nil {
		t.Fatal(err)
	}

	notifier := reloadsignal.Notifier{DataDir: dataDir}
	rounds := admin.NewService(db, db, ingest.NewService(db, importer.Registry{}), notifier)
	rooms := roomconfig.NewService(roomfs.NewStore(filepath.Join(dataDir, "room")), notifier)
	exams := exam.NewService(db, exam.NoUpstream{}, nil)
	health := datacheck.NewService(db, rooms, nil)
	preview := operator.NewPreviewHandler(exams)
	ui := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ui")) }
	h := operator.NewRouter(roomconfighandler.New(rooms), adminhandler.New(rounds, maxUpload), preview, health, ui)
	return &testOperator{t: t, handler: h, dataDir: dataDir, db: db}
}

// request builds a request as the operator app's own UI sends it.
func request(method, path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost:8081"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func (o *testOperator) serve(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	o.handler.ServeHTTP(rec, req)
	return rec
}

func (o *testOperator) json(method, path, body string) *httptest.ResponseRecorder {
	return o.serve(request(method, path, body, map[string]string{"Content-Type": "application/json", "Origin": "http://localhost:8081"}))
}

// upload posts a seating file to /api/rounds/import; headers default to the
// UI's (Origin + X-Operator).
func (o *testOperator) upload(fields map[string]string, filename, content string, headers map[string]string) *httptest.ResponseRecorder {
	return o.uploadTo("/api/rounds/import", fields, filename, content, headers)
}

func (o *testOperator) uploadTo(path string, fields map[string]string, filename, content string, headers map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	if filename != "" {
		fw, _ := mw.CreateFormFile("file", filename)
		io.WriteString(fw, content)
	}
	mw.Close()
	if headers == nil {
		headers = map[string]string{"Origin": "http://localhost:8081", "X-Operator": "1"}
	}
	headers["Content-Type"] = mw.FormDataContentType()
	return o.serve(request(http.MethodPost, path, buf.String(), headers))
}

// reloadSignalled reports whether the .reload marker exists, then removes it.
func (o *testOperator) reloadSignalled() bool {
	path := reloadsignal.Path(o.dataDir)
	_, err := os.Stat(path)
	os.Remove(path)
	return err == nil
}
