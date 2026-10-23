// Command operator manages the site's data: importing seating files,
// renaming and deleting rounds, and editing rooms and seating layouts. It has
// two front ends on the same use cases:
//
//	operator        terminal UI (default); opens no network port
//	operator web    browser UI on OPERATOR_ADDR (127.0.0.1:8081), for the
//	                visual layout editor; reach it through an SSH tunnel
//
// It never runs inside the public server. Changes are picked up by a running
// server through $DATA_DIR/.reload.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/datacheck"
	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer"
	"github.com/openkku/cp-examseat-backend/internal/platform/reloadsignal"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
	roomfs "github.com/openkku/cp-examseat-backend/internal/roomconfig/filesystem"
)

// services are the use cases both front ends drive.
type services struct {
	rounds    *admin.Service
	rooms     *roomconfig.Service
	exams     *exam.Service
	datacheck *datacheck.Service
}

func main() {
	mode := "tui"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode != "tui" && mode != "web" {
		fmt.Fprintln(os.Stderr, "Usage: operator [tui|web]")
		os.Exit(2)
	}

	cfg := config.Load()
	db, err := sqlite.Open(cfg.DatabasePath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	notifier := reloadsignal.Notifier{DataDir: cfg.DataDir}
	var images datacheck.Images
	if cfg.ImageBaseURL == "" {
		images = fileImages{dir: cfg.ImageDir()}
	}
	roomsSvc := roomconfig.NewService(roomfs.NewStore(cfg.RoomDir()), notifier)
	s := services{
		rounds:    admin.NewService(db, db, ingest.NewService(db, importer.Registry{}), notifier),
		rooms:     roomsSvc,
		exams:     exam.NewService(db, exam.NoUpstream{}, nil),
		datacheck: datacheck.NewService(db, roomsSvc, images),
	}

	if mode == "web" {
		runWeb(cfg, s)
		return
	}
	if err := runTUI(cfg, s); err != nil {
		log.Fatal(err)
	}
}
