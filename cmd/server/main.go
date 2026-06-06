/**
* * Server entry point
 */

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jobflow/internal/bootstrap"
	"jobflow/internal/config"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	app, err := bootstrap.New()
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	go app.SSESubscriber.Start(context.Background(), "user:*:jobs")
	go app.RefreshTokenCleaner.Start(context.Background())

	srv := &http.Server{
		Addr:    ":8080",
		Handler: app.Router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	app.Stop()
}
