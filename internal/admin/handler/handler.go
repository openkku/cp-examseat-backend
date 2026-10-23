// Package handler serves the admin API (/api/admin) over HTTP.
package handler

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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// Admin is the administration the handler exposes.
type Admin interface {
	Status() admin.Status
	LastReload() time.Time
	Rounds(ctx context.Context) ([]admin.RoundSummary, error)
	RenameRound(ctx context.Context, round, label string) (string, error)
	DeleteRound(ctx context.Context, round, customID string) error
	ValidateImport(round, customID, roomLayout string) error
	ValidateFileType(ext string) error
	Import(ctx context.Context, req admin.ImportRequest) (int, error)
	Reload(ctx context.Context) error
}

// Handler backs the token-protected /api/admin API used by the frontend's
// admin page.
type Handler struct {
	admin     Admin
	maxUpload int64
}

// New creates a Handler accepting uploads of at most maxUpload bytes.
func New(admin Admin, maxUpload int64) *Handler {
	return &Handler{admin: admin, maxUpload: maxUpload}
}

// Status handles GET /api/admin/status.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	st := h.admin.Status()
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rooms":            st.Rooms,
		"rooms_configured": st.RoomsConfigured,
		"last_reload":      st.LastReload,
		"max_upload_mb":    h.maxUpload >> 20,
	})
}

// Rounds handles GET /api/admin/rounds.
func (h *Handler) Rounds(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.admin.Rounds(r.Context())
	if err != nil {
		log.Printf("Admin: round summaries failed: %v", err)
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	httpx.JSON(w, http.StatusOK, summaries)
}

// RenameRound handles PATCH /api/admin/rounds/{round} with {"label": "..."}.
func (h *Handler) RenameRound(w http.ResponseWriter, r *http.Request) {
	round := chi.URLParam(r, "round")
	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
		httpx.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	label, err := h.admin.RenameRound(r.Context(), round, body.Label)
	if h.failed(w, err, "rename") {
		return
	}
	h.respond(w, map[string]any{"round": round, "label": label})
}

// DeleteRound handles DELETE /api/admin/rounds/{round}[?custom_id=ID]. With
// custom_id only that dataset is removed, otherwise the whole round.
func (h *Handler) DeleteRound(w http.ResponseWriter, r *http.Request) {
	round := chi.URLParam(r, "round")
	customID := r.URL.Query().Get("custom_id")

	err := h.admin.DeleteRound(r.Context(), round, customID)
	if h.failed(w, err, "delete") {
		return
	}
	h.respond(w, map[string]any{"deleted": round, "custom_id": customID})
}

// Import handles POST /api/admin/import (multipart/form-data) with fields
// file, round, display, labels (comma-separated), room_layout and custom_id.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload)
	if err := r.ParseMultipartForm(8 << 20); err != nil { // #nosec G120 -- body capped by MaxBytesReader above
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Error(w, fmt.Sprintf("File too large (max %d MB)", h.maxUpload>>20), http.StatusRequestEntityTooLarge)
			return
		}
		httpx.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	round := strings.TrimSpace(r.FormValue("round"))
	customID := strings.TrimSpace(r.FormValue("custom_id"))
	roomLayout := strings.TrimSpace(r.FormValue("room_layout"))
	if err := h.admin.ValidateImport(round, customID, roomLayout); h.failed(w, err, "import") {
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, "A file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if err := h.admin.ValidateFileType(ext); h.failed(w, err, "import") {
		return
	}

	tmp, err := os.CreateTemp("", "import-*"+ext)
	if err != nil {
		httpx.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		httpx.Error(w, "Invalid upload", http.StatusBadRequest)
		return
	}
	tmp.Close()

	saved, err := h.admin.Import(r.Context(), admin.ImportRequest{
		FilePath:   tmp.Name(),
		Round:      round,
		Display:    strings.TrimSpace(r.FormValue("display")),
		Labels:     domain.ParseLabels(r.FormValue("labels")),
		RoomLayout: roomLayout,
		CustomID:   customID,
	})
	var reloadErr *admin.ReloadError
	if err != nil && !errors.As(err, &reloadErr) {
		log.Printf("Admin: import of %q failed: %v", header.Filename, err)
		// Extraction errors describe the file, not the server, so show them.
		httpx.Error(w, "Import failed: "+strings.ReplaceAll(err.Error(), tmp.Name(), header.Filename), http.StatusUnprocessableEntity)
		return
	}
	if h.failed(w, err, "import") {
		return
	}
	h.respond(w, map[string]any{"round": round, "seats": saved})
}

// Reload handles POST /api/admin/reload.
func (h *Handler) Reload(w http.ResponseWriter, r *http.Request) {
	if h.failed(w, h.admin.Reload(r.Context()), "reload") {
		return
	}
	h.respond(w, map[string]any{})
}

// failed writes the response for a failed admin operation and reports
// whether err was non-nil.
func (h *Handler) failed(w http.ResponseWriter, err error, op string) bool {
	var invalid *admin.InvalidInputError
	var reloadErr *admin.ReloadError
	switch {
	case err == nil:
		return false
	case errors.As(err, &invalid):
		httpx.Error(w, invalid.Message, http.StatusBadRequest)
	case errors.Is(err, domain.ErrRoundNotFound):
		httpx.NotFound(w, "Round not found")
	case errors.As(err, &reloadErr):
		log.Printf("Admin: reload failed: %v", reloadErr.Err)
		httpx.Error(w, "Saved, but reloading failed: "+reloadErr.Err.Error(), http.StatusInternalServerError)
	default:
		log.Printf("Admin: %s failed: %v", op, err)
		httpx.Error(w, "Database Error", http.StatusInternalServerError)
	}
	return true
}

// respond writes body with the time data was last reloaded.
func (h *Handler) respond(w http.ResponseWriter, body map[string]any) {
	body["reloaded_at"] = h.admin.LastReload()
	httpx.JSON(w, http.StatusOK, body)
}
