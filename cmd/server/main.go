package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ahmethakanbesel/youtube-video-summary/internal/middleware"
	"github.com/ahmethakanbesel/youtube-video-summary/internal/transcript"
	"github.com/ahmethakanbesel/youtube-video-summary/pkg/youtube"
)

//go:embed dist/*
var uiAssets embed.FS

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Println("YouTube Video Summary API")

	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Build Date: %s\n", date)

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize packages
	youtubeClient := youtube.NewClient(apiKey, true, logger)
	repo := transcript.NewMemoryRepository(logger)
	svc := transcript.NewService(youtubeClient, repo)
	rtr := transcript.NewRouter(svc, uiAssets)

	// Middleware
	mw := middleware.NewMiddleware(logger)
	handler := mw.Apply(rtr)

	// Server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}

	go func() {
		logger.Info("Starting server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("Server stopped")
}
