package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/operator/tui"
)

// runTUI shows the terminal UI. Log lines (import progress) would garble the
// screen, so they go to $DATA_DIR/operator.log while it runs.
func runTUI(cfg config.Config, s services) error {
	logPath := filepath.Join(cfg.DataDir, "operator.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- fixed name inside DATA_DIR
	if err != nil {
		return err
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	defer log.SetOutput(os.Stderr)

	return tui.Run(s.rounds, s.rooms, filepath.Join(cfg.DataDir, "source"))
}
