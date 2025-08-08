package main

import (
	"log/slog"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"
	"github.com/xcoulon/converse-mcp/pkg/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	url := flag.String("argocd-url", "", "URL of the Argo CD server to query")
	tokenFile := flag.String("argocd-token-file", "", "File with the token to query Argo CD")
	insecure := flag.Bool("insecure", false, "Allow insecure TLS connections")
	flag.Parse()
	token, err := os.ReadFile(*tokenFile)
	if err != nil {
		logger.Error("failed to read token file", "error", err)
		os.Exit(1)
	}
	b := argocdmcp.NewHTTPClient(*url, string(token), *insecure)
	srv := argocdmcp.NewServer(logger, b)
	srv.Start(server.StdioChannel)
	logger.Info("server started")
	if err := srv.Wait(); err != nil {
		logger.Error("failed to wait for server", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("server stopped")
}
