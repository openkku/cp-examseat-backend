package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	adminhandler "github.com/openkku/cp-examseat-backend/internal/admin/handler"
	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/operator"
	roomconfighandler "github.com/openkku/cp-examseat-backend/internal/roomconfig/handler"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

type fileImages struct {
	dir string
}

func (f fileImages) Exists(webPath string) bool {
	rel := strings.TrimPrefix(webPath, "/room/image/")
	rel = strings.TrimPrefix(rel, "/")
	p := filepath.Join(f.dir, filepath.Clean(rel))
	cleanDir := filepath.Clean(f.dir)
	if !strings.HasPrefix(p, cleanDir) {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// runWeb serves the browser UI on OPERATOR_ADDR (loopback by default) until
// interrupted.
func runWeb(cfg config.Config, s services) {
	router := operator.NewRouter(
		roomconfighandler.New(s.rooms),
		adminhandler.New(s.rounds, cfg.MaxUploadBytes),
		operator.NewPreviewHandler(s.exams),
		s.datacheck,
		serveEmbeddedFrontend,
	)

	// Loopback only: the app changes data and has no authentication. In
	// Docker it listens on 0.0.0.0 inside the container, published only on
	// the host's 127.0.0.1 (see the deployment's operator service).
	addr := os.Getenv("OPERATOR_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("🚀 Operator app running on http://%s\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Operator app ListenAndServe error: %v", err)
		}
	}()

	sig := <-stop
	log.Printf("Received signal: %v. Shutting down the operator app...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Operator app shutdown forced: %v", err)
	}
}

// serveEmbeddedFrontend serves the built editor UI, falling back to
// index.html for client-side routes and to a help page when it is not built.
func serveEmbeddedFrontend(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" || path == "" {
		path = "index.html"
	}

	path = strings.TrimPrefix(filepath.Clean(path), "/")

	fileBytes, err := frontendDist.ReadFile(filepath.Join("frontend/dist", path))
	if err != nil {
		fileBytes, err = frontendDist.ReadFile("frontend/dist/index.html")
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(missingBuildPage))
			return
		}
		path = "index.html"
	}

	if mimeType := mime.TypeByExtension(filepath.Ext(path)); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	w.Write(fileBytes) // #nosec G705 -- embedded build assets, not user input
}

const missingBuildPage = `
<!DOCTYPE html>
<html>
<head>
	<title>Operator</title>
	<meta charset="UTF-8">
</head>
<body style="font-family: sans-serif; padding: 2rem; background: #0f172a; color: #f8fafc; text-align: center;">
	<h1>Operator</h1>
	<p>Frontend build assets not found.</p>
	<p>During development, please run the Vite dev server inside <code>cmd/operator/frontend</code>:</p>
	<pre style="background: #1e293b; padding: 1rem; border-radius: 8px; display: inline-block; text-align: left;">
cd cmd/operator/frontend
npm run dev</pre>
	<p>For production, build the assets first:</p>
	<pre style="background: #1e293b; padding: 1rem; border-radius: 8px; display: inline-block; text-align: left;">
cd cmd/operator/frontend
npm run build</pre>
</body>
</html>
`
