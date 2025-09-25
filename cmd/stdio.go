package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// stdioCmd represents the stdio command
var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start the Argo CD MCP server using stdio",
	Run: func(cmd *cobra.Command, _ []string) {
		lvl := new(slog.LevelVar)
		lvl.Set(slog.LevelInfo)
		logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
			Level: lvl,
		}))
		if debug {
			lvl.Set(slog.LevelDebug)
			logger.Debug("debug mode enabled")
		}
		logger.Info("starting Argo CD MCP server using stdio", "url", url, "token", token[:10]+"...", "insecure", insecure, "debug", debug)
		cl := argocdmcp.NewArgoCDClient(url, token, insecure)
		srv := argocdmcp.NewServer(logger, cl)
		t := &mcp.LoggingTransport{
			Transport: &mcp.StdioTransport{},
			Writer:    cmd.ErrOrStderr(),
		}
		if err := srv.Run(context.Background(), t); err != nil {
			logger.Error("failed to serve on stdio", "error", err.Error())
			os.Exit(1)
		}
		logger.Info("bye!")
	},
}

func init() {
	rootCmd.AddCommand(stdioCmd)
	stdioCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
