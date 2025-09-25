package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"

	"github.com/spf13/cobra"
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
		cl := argocdmcp.NewArgoCDClient(url, token, insecure)
		srv := argocdmcp.NewServer(logger, cl)
		// if err := srv.Run(context.Background(), &mcp.StreamableServerTransport{}); err != nil {
		// 	logger.Error("failed to start server", "error", err.Error())
		// 	os.Exit(1)
		// }
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return srv
		}, nil)
		server := &http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		if err := server.ListenAndServe(); err != nil {
			logger.Error("failed to start server", "error", err.Error())
			os.Exit(1)
		}
	},
}

var port int

func init() {
	rootCmd.AddCommand(httpCmd)
	httpCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
}
