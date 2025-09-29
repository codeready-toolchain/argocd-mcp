package e2etests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------------------------------
// Note: make sure you ran `task install` before running this test
// ------------------------------------------------------------------------------------------------

func TestServer(t *testing.T) {

	t.Run("stdio", func(t *testing.T) {
		// given
		ctx := context.Background()
		cmd := newServerCmd(ctx, "stdio")
		cl := mcp.NewClient(&mcp.Implementation{Name: "e2e-test-client", Version: "v1.0.0"}, nil)
		session, err := cl.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		require.NoError(t, err)
		defer session.Close()
		params := &mcp.CallToolParams{
			Name: "unhealthyApplications",
		}

		// when
		res, err := session.CallTool(ctx, params)

		// then
		require.NoError(t, err)
		require.IsType(t, []mcp.Content{}, res.Content)
		require.IsType(t, &mcp.TextContent{}, res.Content[0]) // no need to verify the content, it's already tested in the server_test.go file
	})

	t.Run("http", func(t *testing.T) {
		// given
		ctx := context.Background()
		cmd := newServerCmd(ctx, "http")
		defer func() {
			if err := cmd.Process.Kill(); err != nil {
				t.Errorf("failed to kill command: %v", err)
			}
		}()
		go func() {
			err := cmd.Run()
			t.Errorf("failed to run command: %v", err)
		}()
		cl := mcp.NewClient(&mcp.Implementation{Name: "e2e-test-client", Version: "v1.0.0"}, nil)
		transport := &mcp.StreamableClientTransport{
			Endpoint: fmt.Sprintf("http://localhost:%s", os.Getenv("ARGOCD_MCP_PORT")),
		}
		time.Sleep(5 * time.Second)
		session, err := cl.Connect(ctx, transport, nil)
		require.NoError(t, err)
		defer session.Close()

		// when
		res, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "unhealthyApplications",
		})

		// then
		require.NoError(t, err)
		require.IsType(t, []mcp.Content{}, res.Content)
		require.IsType(t, &mcp.TextContent{}, res.Content[0]) // no need to verify the content, it's already tested in the server_test.go file
	})
}

func newServerCmd(ctx context.Context, transport string) *exec.Cmd {
	argocdURL := os.Getenv("ARGOCD_SERVER_URL")
	argocdToken := os.Getenv("ARGOCD_SERVER_TOKEN")
	argocdMCPPort := os.Getenv("ARGOCD_MCP_PORT")
	return exec.CommandContext(ctx, "argocd-mcp", "--transport", transport, "--argocd-url", argocdURL, "--argocd-token", argocdToken, "--debug", "true", "--port", argocdMCPPort)
}
