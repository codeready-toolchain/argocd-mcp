package argocdmcp_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/codeready-toolchain/argocd-mcp/internal/argocdmcp"
	testresources "github.com/codeready-toolchain/argocd-mcp/test/resources"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/h2non/gock"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestServer(t *testing.T) {
	// given
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	// Argo CD client (to be intercepted by gock)
	argoCl := argocdmcp.NewArgoCDClient("https://argocd-server", "secure-token", false)

	testdata := []struct {
		name string
		init func(*testing.T) (*mcp.ClientSession, func())
	}{
		{
			name: "stdio-ok",
			init: func(t *testing.T) (*mcp.ClientSession, func()) {
				mcpSrv := argocdmcp.NewServer(logger, argoCl)
				// Create a client.
				cl := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
				// Connect the server and client.
				t1, t2 := mcp.NewInMemoryTransports()
				_, err := mcpSrv.Connect(ctx, t1, nil)
				require.NoError(t, err)
				session, err := cl.Connect(ctx, t2, nil)
				require.NoError(t, err)
				return session, func() {
					session.Close()
				}
			},
		},
		{
			name: "http",
			init: func(t *testing.T) (*mcp.ClientSession, func()) {
				var err error
				mcpSrv := argocdmcp.NewServer(logger, argoCl)
				handler := mcp.NewStreamableHTTPHandler(
					func(_ *http.Request) *mcp.Server {
						return mcpSrv
					},
					&mcp.StreamableHTTPOptions{},
				)
				httpServer := httptest.NewServer(handler)
				gock.EnableNetworking() // so we can call the MCP server through the HTTP client
				gock.NetworkingFilter(func(req *http.Request) bool {
					return req.URL.String() == httpServer.URL // allow network calls to the HTTP/MCP server (if any)
				})
				httpSession, err := newHTTPSession(ctx, httpServer.URL)
				require.NoError(t, err)
				return httpSession,
					func() {
						httpSession.Close()
						httpServer.Close()
						gock.DisableNetworkingFilters()
					}
			},
		},
	}

	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			t.Run("call/unhealthyApplications/ok", func(t *testing.T) {
				gock.New("https://argocd-server").
					Get("/api/v1/applications").
					MatchHeader("Authorization", "Bearer secure-token").
					Reply(200).
					BodyString(testresources.ApplicationsStr)
				defer gock.Off() // disable HTTP interceptor after test execution
				session, closeFunc := td.init(t)
				defer closeFunc()

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

			t.Run("call/unhealthyApplications/argocd-unreachable", func(t *testing.T) {
				// given
				session, closeFunc := td.init(t)
				defer closeFunc()

				// when
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
					Name: "unhealthyApplications",
				})

				// then
				require.NoError(t, err)
				assert.True(t, result.IsError)
			})

			t.Run("call/unhealthyApplicationResources/ok", func(t *testing.T) {
				// given
				gock.New("https://argocd-server").
					Get("/api/v1/applications").
					MatchParam("name", "example").
					MatchHeader("Authorization", "Bearer secure-token").
					Reply(200).
					BodyString(testresources.ExampleApplicationStr)
				defer gock.Off() // disable HTTP interceptor after test execution
				session, closeFunc := td.init(t)
				defer closeFunc()

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

			t.Run("call/unhealthyApplicationResources/argocd-unreachable", func(t *testing.T) {
				// given
				session, closeFunc := td.init(t)
				defer closeFunc()

				// when
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
					Name: "unhealthyApplicationResources",
					Arguments: map[string]any{
						"name": "example",
					},
				})

				// then
				require.NoError(t, err)
				assert.True(t, result.IsError)
			})
		})
	}
}

func newHTTPSession(ctx context.Context, srvURL string) (*mcp.ClientSession, error) {
	// Create a client and connect it to the server using our StreamableClientTransport.
	// Check that all requests honor a custom client.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(srvURL)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "test-cookie", Value: "test-value"}})
	httpClient := &http.Client{Jar: jar}
	transport := &mcp.StreamableClientTransport{
		Endpoint:   srvURL,
		HTTPClient: httpClient,
	}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "argocd-mcp-test-client",
		Version: "0.1",
	}, &mcp.ClientOptions{
		CreateMessageHandler: func(context.Context, *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
			return &mcp.CreateMessageResult{Model: "aModel", Content: &mcp.TextContent{}}, nil
		},
	})
	return client.Connect(ctx, transport, nil)
}
