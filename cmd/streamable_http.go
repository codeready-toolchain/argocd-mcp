package cmd

import (
	"log/slog"
	"os"

	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"

	"github.com/spf13/cobra"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"
)

// stdioCmd represents the stdio command
var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Start the Argo CD MCP server using Streamable HTTP",
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
		logger.Info("starting Argo CD MCP server using streamable HTTP", "url", url, "token", token[:10]+"...", "insecure", insecure, "debug", debug, "port", port)
		cl := argocdmcp.NewHTTPClient(url, token, insecure)
		router := argocdmcp.NewRouter(logger, cl)
		srv := mcpserver.NewStreamableHTTPServer(logger, router, port)
		srv.Start()
		if err := srv.Wait(); err != nil {
			logger.Error("failed to wait for server", "error", err.Error())
			os.Exit(1)
		}
	},
}

var port int

func init() {
	rootCmd.AddCommand(httpCmd)
	httpCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
}
