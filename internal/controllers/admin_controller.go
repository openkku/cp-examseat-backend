package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// Reloader refreshes the in-memory state (rounds, rooms, statistics, caches)
// after data changes.
type Reloader interface {
	Reload(ctx context.Context) error
	LastReload() time.Time
}

// AdminController backs the token-protected /api/admin API used by the
// frontend's admin page: importing seating files, managing rounds, reloading
// and backups.
type AdminController struct {
	repo      models.ExamRepository
	ingest    *services.IngestService
	backups   *services.BackupService
	rooms     *services.RoomService
	reloader  Reloader
	maxUpload int64
}

// NewAdminController creates an AdminController.
func NewAdminController(repo models.ExamRepository, ingest *services.IngestService, backups *services.BackupService, rooms *services.RoomService, reloader Reloader, maxUpload int64) *AdminController {
	return &AdminController{repo: repo, ingest: ingest, backups: backups, rooms: rooms, reloader: reloader, maxUpload: maxUpload}
}

// idPattern restricts round and dataset IDs to safe identifiers.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)

var importExtensions = map[string]bool{".pdf": true, ".xlsx": true, ".xls": true, ".json": true}

// Status handles GET /api/admin/status.
func (c *AdminController) Status(w http.ResponseWriter, r *http.Request) {
	backups, err := c.backups.List()
	if err != nil {
		log.Printf("Admin: listing backups failed: %v", err)
		backups = []services.BackupInfo{}
	}
	views.JSON(w, http.StatusOK, map[string]any{
		"rooms":            c.rooms.Count(),
		"rooms_configured": c.rooms.ConfiguredCount(),
		"last_reload":      c.reloader.LastReload(),
		"backups_enabled":  c.backups.Enabled(),
		"backups":          backups,
		"max_upload_mb":    c.maxUpload >> 20,
	})
}

// Rounds handles GET /api/admin/rounds.
func (c *AdminController) Rounds(w http.ResponseWriter, r *http.Request) {
	summaries, err := c.repo.RoundSummaries(r.Context())
	if err != nil {
		log.Printf("Admin: round summaries failed: %v", err)
		views.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	views.JSON(w, http.StatusOK, summaries)
}

// RenameRound handles PATCH /api/admin/rounds/{round} with {"label": "..."}.
func (c *AdminController) RenameRound(w http.ResponseWriter, r *http.Request) {
	round := chi.URLParam(r, "round")
	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		views.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(body.Label)
	if !idPattern.MatchString(round) || label == "" || len(label) > 200 {
		views.Error(w, "A round ID and a label of at most 200 characters are required", http.StatusBadRequest)
		return
	}

	err := c.repo.SetRoundLabel(r.Context(), round, label)
	if errors.Is(err, models.ErrRoundNotFound) {
		views.NotFound(w, "Round not found")
		return
	}
	if err != nil {
		log.Printf("Admin: rename failed: %v", err)
		views.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	c.reloadAndRespond(w, r, map[string]any{"round": round, "label": label})
}

// DeleteRound handles DELETE /api/admin/rounds/{round}[?custom_id=ID]. With
// custom_id only that dataset is removed, otherwise the whole round.
func (c *AdminController) DeleteRound(w http.ResponseWriter, r *http.Request) {
	round := chi.URLParam(r, "round")
	customID := r.URL.Query().Get("custom_id")
	if !idPattern.MatchString(round) || (customID != "" && !idPattern.MatchString(customID)) {
		views.Error(w, "Invalid round or dataset ID", http.StatusBadRequest)
		return
	}

	if !c.roundExists(r.Context(), round) {
		views.NotFound(w, "Round not found")
		return
	}
	c.safetyBackup(r.Context())

	var err error
	if customID != "" {
		err = c.repo.PurgeCustomDataset(r.Context(), round, customID)
	} else {
		err = c.repo.DeleteRound(r.Context(), round)
	}
	if errors.Is(err, models.ErrRoundNotFound) {
		views.NotFound(w, "Round not found")
		return
	}
	if err != nil {
		log.Printf("Admin: delete failed: %v", err)
		views.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	c.reloadAndRespond(w, r, map[string]any{"deleted": round, "custom_id": customID})
}

// Import handles POST /api/admin/import (multipart/form-data) with fields
// file, round, display, labels (comma-separated), room_layout and custom_id.
func (c *AdminController) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, c.maxUpload)
	if err := r.ParseMultipartForm(8 << 20); err != nil { // #nosec G120 -- body capped by MaxBytesReader above
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			views.Error(w, fmt.Sprintf("File too large (max %d MB)", c.maxUpload>>20), http.StatusRequestEntityTooLarge)
			return
		}
		views.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	round := strings.TrimSpace(r.FormValue("round"))
	customID := strings.TrimSpace(r.FormValue("custom_id"))
	roomLayout := strings.TrimSpace(r.FormValue("room_layout"))
	if !idPattern.MatchString(round) {
		views.Error(w, "Round ID is required (letters, digits, _ and - only)", http.StatusBadRequest)
		return
	}
	if (customID != "" && !idPattern.MatchString(customID)) || (roomLayout != "" && !idPattern.MatchString(roomLayout)) {
		views.Error(w, "Invalid custom ID or room layout", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		views.Error(w, "A file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !importExtensions[ext] {
		views.Error(w, "Unsupported file type (use .xlsx, .xls, .pdf or .json)", http.StatusBadRequest)
		return
	}

	tmp, err := os.CreateTemp("", "import-*"+ext)
	if err != nil {
		views.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		views.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	tmp.Close()

	display := strings.TrimSpace(r.FormValue("display"))
	if display == "" {
		display = models.DefaultRoundLabel(round)
	}

	c.safetyBackup(r.Context())

	saved, err := c.ingest.Ingest(r.Context(), services.IngestOptions{
		FilePath:    tmp.Name(),
		RoundID:     round,
		DisplayName: display,
		Labels:      models.ParseLabels(r.FormValue("labels")),
		RoomLayout:  roomLayout,
		CustomID:    customID,
	})
	if err != nil {
		log.Printf("Admin: import of %q failed: %v", header.Filename, err)
		// Extraction errors describe the file, not the server, so show them.
		views.Error(w, "Import failed: "+strings.ReplaceAll(err.Error(), tmp.Name(), header.Filename), http.StatusUnprocessableEntity)
		return
	}
	c.reloadAndRespond(w, r, map[string]any{"round": round, "seats": saved})
}

// Reload handles POST /api/admin/reload.
func (c *AdminController) Reload(w http.ResponseWriter, r *http.Request) {
	c.reloadAndRespond(w, r, map[string]any{})
}

// CreateBackup handles POST /api/admin/backups.
func (c *AdminController) CreateBackup(w http.ResponseWriter, r *http.Request) {
	if !c.backups.Enabled() {
		views.Error(w, "Backups are not configured (set BACKUP_DIR)", http.StatusConflict)
		return
	}
	info, err := c.backups.Snapshot(r.Context())
	if err != nil {
		log.Printf("Admin: backup failed: %v", err)
		views.Error(w, "Backup failed", http.StatusInternalServerError)
		return
	}
	views.JSON(w, http.StatusCreated, info)
}

// DownloadBackup handles GET /api/admin/backup, streaming a fresh snapshot.
func (c *AdminController) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	name := "exams-" + time.Now().UTC().Format("20060102-150405") + ".db"
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if err := c.backups.WriteTo(r.Context(), w); err != nil {
		log.Printf("Admin: backup download failed: %v", err)
	}
}

func (c *AdminController) roundExists(ctx context.Context, round string) bool {
	rounds, err := c.repo.GetRounds(ctx)
	if err != nil {
		return true // let the delete itself report the database error
	}
	for _, r := range rounds {
		if r.ID == round {
			return true
		}
	}
	return false
}

// safetyBackup snapshots the database before a destructive change when
// backups are configured. Failures are logged, not fatal.
func (c *AdminController) safetyBackup(ctx context.Context) {
	if !c.backups.Enabled() {
		return
	}
	if _, err := c.backups.Snapshot(ctx); err != nil {
		log.Printf("⚠️ Safety backup before admin change failed: %v", err)
	}
}

func (c *AdminController) reloadAndRespond(w http.ResponseWriter, r *http.Request, body map[string]any) {
	if err := c.reloader.Reload(r.Context()); err != nil {
		log.Printf("Admin: reload failed: %v", err)
		views.Error(w, "Saved, but reloading failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	body["reloaded_at"] = c.reloader.LastReload()
	views.JSON(w, http.StatusOK, body)
}
