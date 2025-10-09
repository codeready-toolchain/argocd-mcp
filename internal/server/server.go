package server

import (
	"context"
	"log/slog"

	"github.com/codeready-toolchain/argocd-mcp/internal/argocd"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func New(logger *slog.Logger, cl argocd.Client) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "argocd-mcp",
			Version: "0.1",
		},
		&mcp.ServerOptions{
			InitializedHandler: func(_ context.Context, ir *mcp.InitializedRequest) {
				logger.Debug("initialized", "session_id", ir.Session.ID())
			},
		},
	)

	s.AddPrompt(argocd.UnhealthyResourcesPrompt, argocd.UnhealthyApplicationResourcesPromptHandle(logger, cl))
	mcp.AddTool(s, argocd.UnhealthyApplicationsTool, argocd.UnhealthyApplicationsToolHandle(logger, cl))
	mcp.AddTool(s, argocd.UnhealthyApplicationResourcesTool, argocd.UnhealthyApplicationResourcesToolHandle(logger, cl))
	return s
}
