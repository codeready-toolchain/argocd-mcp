package argocdmcp

import (
	"log/slog"

	"github.com/creachadair/jrpc2"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"
)

func NewServer(logger *slog.Logger, cl HTTPClient) *jrpc2.Server {
	mux := mcpserver.NewMux("argocd-mcp", "0.1", logger).
		WithPrompt(UnhealthyResourcesPrompt, UnhealthyApplicationResourcesPromptHandle(logger, cl)).
		WithTool(UnhealthyApplicationsTool, UnhealthyApplicationsToolHandle(logger, cl)).
		WithTool(UnhealthyApplicationResourcesTool, UnhealthyApplicationResourcesToolHandle(logger, cl)).
		Build()
	return mcpserver.NewStdioServer(mux, logger)
}
