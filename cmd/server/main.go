// Command server runs the CP Exam Seat JSON API.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/app"
	"github.com/openkku/cp-examseat-backend/internal/config"
)

func main() {
	cfg := config.Load()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Handler,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    16 << 10, // URLs are short; cap request line + headers at 16 KiB
	}

	// Create channel to listen for interrupt/termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🔥 API server running on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ListenAndServe error: %v", err)
		}
	}()

	// Block until a signal is received
	sig := <-stop
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	// Context with timeout to allow active requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown forced: %v", err)
	} else {
		log.Println("Server exited cleanly")
	}
}
