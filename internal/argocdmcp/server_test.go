package argocdmcp_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/xcoulon/argocd-mcp/internal/argocdmcp"

	mcpapi "github.com/xcoulon/converse-mcp/pkg/api"
	mcpchannel "github.com/xcoulon/converse-mcp/pkg/channel"
	mcpclient "github.com/xcoulon/converse-mcp/pkg/client"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:embed argocd-applications-example.json
var exampleApplicationsStr string

//go:embed argocd-applications.json
var applicationsStr string

func TestServer(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Argo CD client (to be intercepted by gock)
	argocdCl := argocdmcp.NewHTTPClient("https://argocd-server", "secure-token", false)

	router := argocdmcp.NewRouter(logger, argocdCl)

	// stdio server
	c2s, s2c := mcpchannel.Direct()
	stdioCl := mcpclient.NewFromChannel(c2s)
	stdioSrv := mcpserver.NewStdioServer(logger, router)
	stdioSrv.Start(s2c)
	defer func() {
		stdioSrv.Stop()
		if err := stdioCl.Close(); err != nil {
			t.Errorf("failed to close client: %v", err)
		}
	}()

	// streamable http server
	httpSrv := mcpserver.NewStreamableHTTPServer(logger, router, 8080)
	httpSrv.Start()
	httpCl := mcpclient.NewFromURL(httpSrv.Addr())
	defer func() {
		if err := httpSrv.Stop(); err != nil {
			t.Errorf("failed to stop http server: %v", err)
		}
		if err := httpCl.Close(); err != nil {
			t.Errorf("failed to close client: %v", err)
		}
	}()

	testdata := []struct {
		name string
		cl   *mcpclient.Client
	}{
		{
			name: "stdio",
			cl:   stdioCl,
		},
		{
			name: "http",
			cl:   httpCl,
		},
	}

	for _, test := range testdata {
		t.Run(test.name, func(t *testing.T) {
			t.Run("call/unhealthyApplications", func(t *testing.T) {
				// given
				gock.Intercept()
				gock.New("https://argocd-server").
					Get("/api/v1/applications").
					MatchHeader("Authorization", "Bearer secure-token").
					Reply(200).
					BodyString(applicationsStr)
				gock.Observe(gock.DumpRequest)
				defer gock.Off() // Flush pending mocks after test execution

				// when
				resp, err := stdioCl.Call(context.Background(), "tools/call", mcpapi.CallToolRequestParams{
					Name: "unhealthyApplications",
				})

				// then
				require.NoError(t, err)
				callResult := &mcpapi.CallToolResult{}
				err = json.Unmarshal([]byte(resp.ResultString()), callResult)
				require.NoError(t, err)

				// verify the `text` result
				require.IsType(t, map[string]any{}, callResult.Content[0])
				actualTextContent := mcpapi.TextContent{}
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(callResult.Content[0].(map[string]any), &actualTextContent)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"a-progressing-application", "another-progressing-application", "a-degraded-application", "another-degraded-application"}, strings.Split(actualTextContent.Text, ", "))

				// verify the `structured` content
				require.IsType(t, map[string]any{}, callResult.StructuredContent)
				t.Logf("callResult.StructuredContent: %v", callResult.StructuredContent)
				actualStructuredContent := map[string]any{}
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(callResult.StructuredContent, &actualStructuredContent)
				require.NoError(t, err)
				assert.Equal(t,
					map[string]any{
						"progressing": []any{"a-progressing-application", "another-progressing-application"},
						"degraded":    []any{"a-degraded-application", "another-degraded-application"}},
					actualStructuredContent)
			})
			t.Run("call/unhealthyApplicationResources", func(t *testing.T) {
				// given
				gock.Intercept()
				gock.New("https://argocd-server").
					Get("/api/v1/applications").
					MatchParam("name", "example").
					MatchHeader("Authorization", "Bearer secure-token").
					Reply(200).
					BodyString(exampleApplicationsStr)
				gock.Observe(gock.DumpRequest)
				defer gock.Off() // Flush pending mocks after test execution

				// when
				resp, err := stdioCl.Call(context.Background(), "tools/call", mcpapi.CallToolRequestParams{
					Name:      "unhealthyApplicationResources",
					Arguments: map[string]any{"name": "example"},
				})

				// then
				require.NoError(t, err)
				callResult := &mcpapi.CallToolResult{}
				err = json.Unmarshal([]byte(resp.ResultString()), callResult)
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
		})
	}
}
