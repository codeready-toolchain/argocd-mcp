package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"
	"github.com/xcoulon/converse-mcp/pkg/server"
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
		b := argocdmcp.NewHTTPClient(url, token, insecure)
		srv := argocdmcp.NewServer(logger, b)
		srv.Start(server.StdioChannel)
		logger.Info("server started")
		if err := srv.Wait(); err != nil {
			logger.Error("failed to wait for server", "error", err.Error())
			os.Exit(1)
		}
		logger.Info("server stopped")
	},
}

func init() {
	rootCmd.AddCommand(stdioCmd)
	stdioCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
