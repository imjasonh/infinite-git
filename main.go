package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/imjasonh/infinite-git/internal/repo"
	"github.com/imjasonh/infinite-git/internal/server"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Port     string `env:"PORT,default=8080"`
	RepoPath string `env:"REPO_PATH,default=./infinite-repo"`
}{})

func main() {
	// clog/gcp/init automatically sets up the logger

	// Initialize repository
	slog.Info("initializing repository", "env", env)
	gitRepo, err := repo.New(env.RepoPath)
	if err != nil {
		slog.Error("failed to initialize repository", "error", err)
		os.Exit(1)
	}

	// Create server
	srv := server.New(gitRepo)

	// Set up HTTP server
	httpServer := &http.Server{
		Addr:         ":" + env.Port,
		Handler:      srv.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("starting HTTP server", "port", env.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}
