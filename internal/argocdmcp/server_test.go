package argocdmcp_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"
	mcpapi "github.com/xcoulon/converse-mcp/pkg/api"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:embed argocd-applications-example.json
var exampleApplicationsStr string

func TestServer(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	c2s, s2c := channel.Direct()
	cl := jrpc2.NewClient(c2s, &jrpc2.ClientOptions{})

	srv := argocdmcp.NewServer(logger, argocdmcp.NewHTTPClient("https://argocd-server", "secure-token", false)) // do not change the underlying transport of the default client if the insecure flag is not set (so we can test it with gock)
	srv.Start(s2c)
	// close the streams
	defer func(cl *jrpc2.Client, srv *jrpc2.Server) {
		cl.Close()
		srv.Stop()
	}(cl, srv)

	t.Run("call/unhealthyResources", func(t *testing.T) {
		// given
		gock.Intercept()
		gock.New("https://argocd-server").
			Get("/api/v1/applications"). //TODO: check the token
			MatchParam("name", "example").
			MatchHeader("Authorization", "Bearer secure-token").
			Reply(200).
			BodyString(exampleApplicationsStr)
		defer gock.Off() // Flush pending mocks after test execution

		// when
		resp, err := cl.Call(context.Background(), "tools/call", mcpapi.CallToolRequestParams{
			Name:      "unhealthyResources",
			Arguments: map[string]any{"name": "example"},
		})

		// then
		require.NoError(t, err)
		callResult := &mcpapi.CallToolResult{}
		err = json.Unmarshal([]byte(resp.ResultString()), callResult)
		require.NoError(t, err)

		expectedContent := argocdmcp.UnhealthyResources{
			Resources: []*argocdv3.ResourceResult{
				{
					Group:     "external-secrets.io",
					Version:   "v1beta1",
					Kind:      "ExternalSecret",
					Namespace: "example-ns",
					Name:      "example-secret",
					Status:    "SyncFailed",
					Message:   "resource mapping not found for name: \"example-secret\" namespace: \"example-ns\" from \"/dev/shm/3358522048\": no matches for kind \"ExternalSecret\" in version \"external-secrets.io/v1beta1\"\nensure CRDs are installed first",
					HookPhase: "Failed",
					SyncPhase: "Sync",
				},
			},
		}
		expectedResourcesText, err := json.Marshal(expectedContent)
		require.NoError(t, err)

		// verify the `text` result
		require.IsType(t, map[string]any{}, callResult.Content[0])
		actualTextContent := mcpapi.TextContent{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(callResult.Content[0].(map[string]any), &actualTextContent)
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedResourcesText), actualTextContent.Text)

		// verify the `structured` content
		require.IsType(t, map[string]any{}, callResult.StructuredContent)
		actualStructuredContent := argocdmcp.UnhealthyResources{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(callResult.StructuredContent, &actualStructuredContent)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, actualStructuredContent)
	})
}
