package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/codeready-toolchain/argocd-mcp/internal/argocdmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var transport, url, token, insecureStr string
var insecure, debug bool
var port int

func init() {
	startServerCmd.Flags().StringVar(&url, "argocd-url", "", "Specify the URL of the Argo CD server to query (required)")
	if err := startServerCmd.MarkFlagRequired("argocd-url"); err != nil {
		panic(err)
	}
	startServerCmd.Flags().StringVar(&token, "argocd-token", "", "Specify the token to include in the Authorization header (required)")
	if err := startServerCmd.MarkFlagRequired("argocd-token"); err != nil {
		panic(err)
	}
	startServerCmd.Flags().StringVar(&insecureStr, "insecure", "false", "Allow insecure TLS connections to the Argo CD server")
	startServerCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug mode")
	startServerCmd.Flags().StringVar(&transport, "transport", "http", "Choose between 'stdio' or 'http' transport")
	startServerCmd.Flags().IntVarP(&port, "port", "p", 8080, "Specify the port to listen on when using the 'http' transport")
	startServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := startServerCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// startServerCmd the command to start the Argo CD MCP server
var startServerCmd = &cobra.Command{
	Use:   "argocd-mcp",
	Short: "Start the Argo CD MCP server",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if transport != "stdio" && transport != "http" {
			return fmt.Errorf("invalid transport: choose between 'http' and 'stdio'")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		lvl := new(slog.LevelVar)
		lvl.Set(slog.LevelInfo)
		logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
			Level: lvl,
		}))
		logger.Info("starting the Argo CD MCP server", "transport", transport, "url", url, "insecure", insecure, "debug", debug)
		if debug {
			lvl.Set(slog.LevelDebug)
			logger.Debug("debug mode enabled")
		}
		cl := argocdmcp.NewArgoCDClient(url, token, insecure)
		srv := argocdmcp.NewServer(logger, cl)
		switch transport {
		case "stdio":
			t := &mcp.LoggingTransport{
				Transport: &mcp.StdioTransport{},
				Writer:    cmd.ErrOrStderr(),
			}
			if err := srv.Run(context.Background(), t); err != nil {
				return fmt.Errorf("failed to serve on stdio: %v", err.Error())
			}
		default:
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
				return fmt.Errorf("failed to start server: %v", err.Error())
			}
		}
		return nil
	},
}
