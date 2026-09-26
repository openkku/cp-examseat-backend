// Command room-config runs the room layout editor: a local tool that edits
// $DATA_DIR/room/metadata.json and map/*.json through an embedded web UI.
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

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/controllers"
	"github.com/openkku/cp-examseat-backend/internal/repositories/filesystem"
	"github.com/openkku/cp-examseat-backend/internal/routes"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	store := filesystem.NewRoomConfigStore(config.Load().RoomDir())
	handler := routes.NewRoomConfig(controllers.NewRoomConfigController(store), serveEmbeddedFrontend)

	port := "8081"
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("🚀 Room Config Manager running on http://localhost:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Room Config Manager ListenAndServe error: %v", err)
		}
	}()

	sig := <-stop
	log.Printf("Received signal: %v. Shutting down Room Config Manager gracefully...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Room Config Manager shutdown forced: %v", err)
	} else {
		log.Println("Room Config Manager exited cleanly")
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
	w.Write(fileBytes)
}

const missingBuildPage = `
<!DOCTYPE html>
<html>
<head>
	<title>Room Config Manager</title>
	<meta charset="UTF-8">
</head>
<body style="font-family: sans-serif; padding: 2rem; background: #0f172a; color: #f8fafc; text-align: center;">
	<h1>Room Config Manager</h1>
	<p>Frontend build assets not found.</p>
	<p>During development, please run the Vite dev server inside <code>cmd/room-config/frontend</code>:</p>
	<pre style="background: #1e293b; padding: 1rem; border-radius: 8px; display: inline-block; text-align: left;">
cd cmd/room-config/frontend
npm run dev</pre>
	<p>For production, build the assets first:</p>
	<pre style="background: #1e293b; padding: 1rem; border-radius: 8px; display: inline-block; text-align: left;">
cd cmd/room-config/frontend
npm run build</pre>
</body>
</html>
`
