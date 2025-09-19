package argocdmcp

import (
	"log/slog"

	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"
)

func NewRouter(logger *slog.Logger, cl HTTPClient) mcpserver.Router {
	return mcpserver.NewRouterBuilder("argocd-mcp", "0.1", logger).
		WithPrompt(UnhealthyResourcesPrompt, UnhealthyApplicationResourcesPromptHandle(logger, cl)).
		WithTool(UnhealthyApplicationsTool, UnhealthyApplicationsToolHandle(logger, cl)).
		WithTool(UnhealthyApplicationResourcesTool, UnhealthyApplicationResourcesToolHandle(logger, cl)).
		Build()
}
