package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imjasonh/infinite-git/internal/repo"
	"github.com/imjasonh/infinite-git/internal/server"
)

func main() {
	var (
		addr     = flag.String("addr", ":8080", "HTTP server address")
		repoPath = flag.String("repo", "./infinite-repo", "Path to Git repository")
		logLevel = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Set up logger
	level := slog.LevelInfo
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Initialize repository
	logger.Info("initializing repository", "path", *repoPath)
	gitRepo, err := repo.New(*repoPath)
	if err != nil {
		logger.Error("failed to initialize repository", "error", err)
		os.Exit(1)
	}

	// Create server
	srv := server.New(gitRepo, logger)

	// Set up HTTP server
	httpServer := &http.Server{
		Addr:         *addr,
		Handler:      srv.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", "addr", *addr)
		logger.Info("to clone: git clone http://localhost" + *addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("server stopped")
}
