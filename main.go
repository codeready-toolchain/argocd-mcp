package main

import (
	"log/slog"
	"os"

	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"
	"github.com/xcoulon/argocd-mcp/internal/configuration"
	"github.com/xcoulon/converse-mcp/pkg/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	cfg, err := configuration.New()
	if err != nil {
		logger.Error("failed to load configuration", "error", err.Error())
		os.Exit(1)
	}
	b := argocdmcp.NewHTTPClient(cfg.URL, cfg.Token, cfg.Insecure)
	srv := argocdmcp.NewServer(logger, b)
	srv.Start(server.StdioChannel)
	logger.Info("server started")
	if err := srv.Wait(); err != nil {
		logger.Error("failed to wait for server", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("server stopped")
}
