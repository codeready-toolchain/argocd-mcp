package argocdmcp

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewServer(logger *slog.Logger, cl *ArgoCDClient) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "argocd-mcp",
			Version: "0.1",
		},
		&mcp.ServerOptions{
			InitializedHandler: func(_ context.Context, ir *mcp.InitializedRequest) {
				logger.Info("initialized", "session_id", ir.Session.ID())
			},
		},
	)

	s.AddPrompt(UnhealthyResourcesPrompt, UnhealthyApplicationResourcesPromptHandle(logger, cl))
	mcp.AddTool(s, UnhealthyApplicationsTool, UnhealthyApplicationsToolHandle(logger, cl))
	mcp.AddTool(s, UnhealthyApplicationResourcesTool, UnhealthyApplicationResourcesToolHandle(logger, cl))
	return s
}
