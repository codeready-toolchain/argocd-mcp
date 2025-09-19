package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "argocd-mcp",
	Short: "Start the Argo CD MCP server",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var url, token, insecureStr string
var insecure bool
var debug bool

func init() {
	rootCmd.PersistentFlags().StringVar(&url, "argocd-url", "", "URL of the Argo CD server to query (required)")
	rootCmd.PersistentFlags().StringVar(&token, "argocd-token", "", "The token to include in the Authorization header (required)")
	rootCmd.PersistentFlags().StringVar(&insecureStr, "insecure", "false", "Allow insecure TLS connections to the Argo CD server (default: false)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Debug mode (default: false)")
	rootCmd.PersistentFlags().BoolP("toggle", "t", false, "Help message for toggle (default: false)")
}
