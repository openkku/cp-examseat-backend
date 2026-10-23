// Command migrate imports exam seating files (.pdf, .xlsx, .json) into
// $DATA_DIR/exams.db and renames or deletes rounds. A running server picks
// every change up by itself through $DATA_DIR/.reload.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer"
	"github.com/openkku/cp-examseat-backend/internal/platform/reloadsignal"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	arg1 := os.Args[1]

	switch arg1 {
	case "custom", "--custom":
		runCustomMigrate(os.Args[2:])
		return
	case "rename":
		runRename(os.Args[2:])
		return
	case "delete":
		runDelete(os.Args[2:])
		return
	}

	// Legacy mode: go run ./cmd/migrate <FILE_PATH> <ROUND_ID> [DISPLAY_NAME]
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	filePath := os.Args[1]
	roundID := os.Args[2]

	displayName := domain.DefaultRoundLabel(roundID)
	if len(os.Args) > 3 {
		displayName = os.Args[3]
	}

	if err := importFile(ingest.Options{
		FilePath:    filePath,
		RoundID:     roundID,
		DisplayName: displayName,
	}); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
}

func runCustomMigrate(args []string) {
	customCmd := flag.NewFlagSet("custom", flag.ExitOnError)
	filePath := customCmd.String("file", "", "Path to the exam schedule file (.pdf, .xlsx, .json)")
	roundID := customCmd.String("round", "", "Round ID (e.g., 2_2568)")
	displayName := customCmd.String("display", "", "Round display name")
	labelsStr := customCmd.String("labels", "", "Comma-separated exam labels (e.g. LAB,Lab)")
	roomLayout := customCmd.String("room-layout", "", "Custom room layout key (e.g. CP9421_LAB)")
	customID := customCmd.String("custom-id", "", "Custom dataset ID for scoped purging (e.g. FINAL_2_OOP_LAB_2026)")

	if err := customCmd.Parse(args); err != nil {
		log.Fatalf("Failed to parse custom migration flags: %v", err)
	}

	if *filePath == "" || *roundID == "" {
		fmt.Println("❌ Error: Both --file and --round parameters are required for custom migration.")
		fmt.Println("   Usage: go run ./cmd/migrate custom --file <PATH> --round <ROUND_ID> [--custom-id <ID>] [--labels L1,L2] [--room-layout LAYOUT]")
		os.Exit(1)
	}

	finalDisplayName := *displayName
	if finalDisplayName == "" {
		finalDisplayName = domain.DefaultRoundLabel(*roundID)
	}

	labels := domain.ParseLabels(*labelsStr)

	if err := importFile(ingest.Options{
		FilePath:    *filePath,
		RoundID:     *roundID,
		DisplayName: finalDisplayName,
		Labels:      labels,
		RoomLayout:  *roomLayout,
		CustomID:    *customID,
	}); err != nil {
		log.Fatalf("❌ Custom migration failed: %v", err)
	}
}

func printUsage() {
	fmt.Println("❌ Usage:")
	fmt.Println("   Standard Migration:")
	fmt.Println("     go run ./cmd/migrate <FILE_PATH> <ROUND_ID> [DISPLAY_NAME]")
	fmt.Println("     Example: go run ./cmd/migrate data/source/final_2_2568.xlsx 2_2568 \"Final Exam 2/2568\"")
	fmt.Println("")
	fmt.Println("   Custom Edge-Case Migration:")
	fmt.Println("     go run ./cmd/migrate custom --file <PATH> --round <ROUND_ID> [--custom-id ID] [--labels L1,L2] [--room-layout LAYOUT]")
	fmt.Println("     Example: go run ./cmd/migrate custom --file data/source/custom/Lab_2568_CP421024_OOP_CY_FinalExam_Sec1.pdf --round 2_2568 --custom-id FINAL_2_OOP_LAB_2026 --labels LAB,Lab --room-layout CP9421_LAB")
	fmt.Println("")
	fmt.Println("   Rounds:")
	fmt.Println("     go run ./cmd/migrate rename <ROUND_ID> <LABEL>")
	fmt.Println("     go run ./cmd/migrate delete <ROUND_ID> [--custom-id ID]   (only that dataset with --custom-id)")
	fmt.Println("")
	fmt.Println("   A running server reloads by itself within a few seconds.")
}

// importFile imports one file into the database under $DATA_DIR and asks a
// running server to reload.
func importFile(opts ingest.Options) error {
	cfg := config.Load()
	db, err := sqlite.Open(cfg.DatabasePath())
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := ingest.NewService(db, importer.Registry{}).Ingest(context.Background(), opts); err != nil {
		return err
	}
	return reloadsignal.Touch(cfg.DataDir)
}

// withRounds runs fn with the round management of the database under $DATA_DIR.
func withRounds(fn func(*admin.Service) error) error {
	cfg := config.Load()
	db, err := sqlite.Open(cfg.DatabasePath())
	if err != nil {
		return err
	}
	defer db.Close()
	return fn(admin.NewService(db, db, ingest.NewService(db, importer.Registry{}), reloadsignal.Notifier{DataDir: cfg.DataDir}))
}

func runRename(args []string) {
	if len(args) != 2 {
		fmt.Println("❌ Usage: go run ./cmd/migrate rename <ROUND_ID> <LABEL>")
		os.Exit(1)
	}
	var label string
	err := withRounds(func(s *admin.Service) (err error) {
		label, err = s.RenameRound(context.Background(), args[0], args[1])
		return err
	})
	if err != nil {
		log.Fatalf("❌ Rename failed: %v", err)
	}
	fmt.Printf("✅ Renamed %s to %q\n", args[0], label)
}

func runDelete(args []string) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	customID := deleteCmd.String("custom-id", "", "Delete only this custom dataset of the round")
	// Accept the round before or after the flags.
	var round string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		round, args = args[0], args[1:]
	}
	if err := deleteCmd.Parse(args); err != nil {
		log.Fatalf("Failed to parse delete flags: %v", err)
	}
	if round == "" && deleteCmd.NArg() == 1 {
		round = deleteCmd.Arg(0)
	}
	if round == "" {
		fmt.Println("❌ Usage: go run ./cmd/migrate delete <ROUND_ID> [--custom-id ID]")
		os.Exit(1)
	}

	if err := withRounds(func(s *admin.Service) error {
		return s.DeleteRound(context.Background(), round, *customID)
	}); err != nil {
		log.Fatalf("❌ Delete failed: %v", err)
	}
	if *customID != "" {
		fmt.Printf("✅ Deleted dataset %s of round %s\n", *customID, round)
	} else {
		fmt.Printf("✅ Deleted round %s\n", round)
	}
}
