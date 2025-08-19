package argocdmcp

import (
	"log/slog"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"
)

var StdioChannel = channel.Line(os.Stdin, os.Stdout)

func NewServer(logger *slog.Logger, cl HTTPClient) *jrpc2.Server {
	mux := mcpserver.NewMux("argocd-mcp", "0.1", logger).
		WithTool(UnhealthyResourcesTool, UnhealthyResourcesHandle(cl)).
		Build()
	return mcpserver.NewStdioServer(mux, logger)
}
