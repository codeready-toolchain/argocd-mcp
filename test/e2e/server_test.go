package e2etests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/codeready-toolchain/argocd-mcp/internal/argocdmcp"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

// ------------------------------------------------------------------------------------------------
// Note: make sure you ran `task install` before running this test
// ------------------------------------------------------------------------------------------------

func TestServer(t *testing.T) {

	// start the argocd mock server
	cmd := exec.CommandContext(context.Background(), "argocd-mock")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		if err := cmd.Run(); err != nil {
			t.Errorf("failed to run command: %v", err)
		}
	}()
	defer func() {
		t.Logf("killing the Argo CD mock server: %v", cmd.String())
		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("failed to kill the Argo CD mock server: %v", err)
		}
		t.Logf("killed the Argo CD mock server: %v", cmd.String())
	}()

	testdata := []struct {
		name string
		init func(*testing.T) (*mcp.ClientSession, KillMCPServerFunc)
	}{
		{
			name: "stdio",
			init: newStdioSession,
		},
		{
			name: "http",
			init: newHTTPSession,
		},
	}

	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			// given
			session, killMCPServer := td.init(t)
			defer session.Close()
			defer killMCPServer()

			t.Run("call/unhealthyApplications/ok", func(t *testing.T) {
				// when
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
					Name: "unhealthyApplications",
				})

				// then
				require.NoError(t, err)
				require.False(t, result.IsError)
				// expected content
				expectedContent := map[string]any{
					"degraded":    []any{"a-degraded-application", "another-degraded-application"},
					"progressing": []any{"a-progressing-application", "another-progressing-application"},
				}
				expectedContentText, err := json.Marshal(expectedContent)
				require.NoError(t, err)
				// verify the `text` result
				resultContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.JSONEq(t, string(expectedContentText), resultContent.Text)
				// verify the `structured` content
				require.IsType(t, map[string]any{}, result.StructuredContent)
				actualStructuredContent := map[string]any{}
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.StructuredContent.(map[string]any), &actualStructuredContent)
				require.NoError(t, err)
				assert.Equal(t, expectedContent, actualStructuredContent)
			})

			t.Run("call/unhealthyApplicationResources/ok", func(t *testing.T) {
				// when
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
					Name: "unhealthyApplicationResources",
					Arguments: map[string]any{
						"name": "example",
					},
				})

				// then
				require.NoError(t, err)
				expectedContent := argocdmcp.UnhealthyResources{
					Resources: []argocdv3.ResourceStatus{
						{
							Group:     "apps",
							Version:   "v1",
							Kind:      "StatefulSet",
							Namespace: "example-ns",
							Name:      "example",
							Status:    "Synced",
							Health: &argocdv3.HealthStatus{
								Status:  "Progressing",
								Message: "Waiting for 1 pods to be ready...",
							},
						},
						{
							Group:     "external-secrets.io",
							Version:   "v1beta1",
							Kind:      "ExternalSecret",
							Namespace: "example-ns",
							Name:      "example-secret",
							Status:    "OutOfSync",
							Health: &argocdv3.HealthStatus{
								Status: "Missing",
							},
						},
					},
				}
				expectedResourcesText, err := json.Marshal(expectedContent)
				require.NoError(t, err)

				// verify the `text` result
				resultContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.JSONEq(t, string(expectedResourcesText), resultContent.Text)

				// verify the `structured` content
				require.IsType(t, map[string]any{}, result.StructuredContent)
				actualStructuredContent := argocdmcp.UnhealthyResources{}
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.StructuredContent.(map[string]any), &actualStructuredContent)
				require.NoError(t, err)
				assert.Equal(t, expectedContent, actualStructuredContent)
			})

			t.Run("call/unhealthyApplicationResources/argocd-error", func(t *testing.T) {
				// when
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
					Name: "unhealthyApplicationResources",
					Arguments: map[string]any{
						"name": "example-error",
					},
				})

				// then
				require.NoError(t, err)
				assert.True(t, result.IsError)
			})
		})
	}
}

type KillMCPServerFunc func()

func newStdioSession(t *testing.T) (*mcp.ClientSession, KillMCPServerFunc) {
	ctx := context.Background()
	cmd := newServerCmd(ctx, "stdio")
	cl := mcp.NewClient(&mcp.Implementation{Name: "e2e-test-client", Version: "v1.0.0"}, nil)
	session, err := cl.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	require.NoError(t, err)
	return session, func() {
		// nothing to do
	}
}

func newHTTPSession(t *testing.T) (*mcp.ClientSession, KillMCPServerFunc) {
	ctx := context.Background()
	cmd := newServerCmd(ctx, "http")
	go func() {
		t.Logf("starting the MCP server: %v", cmd.String())
		if err := cmd.Run(); err != nil {
			exitErr := &exec.ExitError{}
			// Ignore expected exit error when the process is killed in teardown.
			if !errors.As(err, &exitErr) {
				t.Errorf("failed to run command: %v", err)
			}
		}
	}()
	time.Sleep(5 * time.Second)
	cl := mcp.NewClient(&mcp.Implementation{Name: "e2e-test-client", Version: "v1.0.0"}, nil)
	session, err := cl.Connect(ctx, &mcp.StreamableClientTransport{
		MaxRetries: 5,
		Endpoint:   fmt.Sprintf("http://%s", os.Getenv("MCP_SERVER_LISTEN")),
	}, nil)
	require.NoError(t, err)
	return session, func() {
		t.Logf("killing the MCP server")
		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("failed to kill the MCP server: %v", err)
		}
		t.Logf("killed the MCP server")
	}
}

func newServerCmd(ctx context.Context, transport string) *exec.Cmd {
	mcpServerListen := os.Getenv("MCP_SERVER_LISTEN")
	mcpServerDebug := os.Getenv("MCP_SERVER_DEBUG")
	argocdURL := fmt.Sprintf("http://%s", os.Getenv("ARGOCD_SERVER_LISTEN"))
	argocdToken := os.Getenv("ARGOCD_SERVER_TOKEN")
	return exec.CommandContext(ctx,
		"argocd-mcp",
		"--transport", transport,
		"--listen", mcpServerListen,
		"--debug", mcpServerDebug,
		"--argocd-url", argocdURL,
		"--argocd-token", argocdToken,
	)
}
