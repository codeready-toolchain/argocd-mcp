package argocdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/sync/common"
	mcpapi "github.com/xcoulon/converse-mcp/pkg/api"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"
	"k8s.io/apimachinery/pkg/runtime"
)

var UnhealthyResourcesPrompt = mcpapi.NewPrompt("argocd-unhealthy-application-resources").
	WithArgument("name", "the name of the name to get details of", "", true)

func UnhealthyResourcesPromptHandle(cl HTTPClient) mcpserver.PromptHandleFunc {
	return func(ctx context.Context, logger *slog.Logger, params mcpapi.GetPromptRequestParams) (any, error) {
		app, ok := params.Arguments["name"]
		if !ok {
			return nil, fmt.Errorf("'name' not found in arguments or not a string")
		}
		unhealthyResources, err := getUnhealthyResources(ctx, logger, cl, app)
		if err != nil {
			return nil, err
		}
		unhealthyResourcesText, err := json.Marshal(unhealthyResources)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unhealthy resources to text: %w", err)
		}
		result := &mcpapi.GetPromptResult{
			Description: mcpapi.StringPtr("The unhealthy resources of the Argo CD Application prompt"),
			Messages: []mcpapi.PromptMessage{
				{
					Role: mcpapi.RoleUser,
					Content: mcpapi.TextContent{
						Type: "text",
						Text: string(unhealthyResourcesText),
					},
				},
			},
		}
		if logger.Enabled(ctx, slog.LevelDebug) {
			logger.DebugContext(ctx, "returned 'prompt/get' response", "content", result)
		}
		return result, nil
	}
}

var UnhealthyResourcesTool = mcpapi.NewTool("unhealthyResources").
	WithTitle("get unhealthy resources of an Argo CD Application").
	WithDestructiveHint(false).
	WithReadOnlyHint(true).
	WithInputProperty("name", mcpapi.String, "the name of the Argo CD Application to get details of", true)
	// WithOutputProperty("resources", mcpapi.Array, "the unhealthy resources of the Argo CD Application", true)

func UnhealthyResourcesToolHandle(cl HTTPClient) mcpserver.ToolHandleFunc {
	return func(ctx context.Context, logger *slog.Logger, params mcpapi.CallToolRequestParams) (any, error) {
		app, ok := params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("'name' not found in arguments or not a string")
		}
		unhealthyResources, err := getUnhealthyResources(ctx, logger, cl, app)
		if err != nil {
			return nil, err
		}
		unhealthyResourcesText, err := json.Marshal(unhealthyResources)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unhealthy resources to 'text' content: %w", err)
		}
		unhealthyResourcesStructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(unhealthyResources)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unhealthy resources to 'structured' content: %w", err)
		}
		result := &mcpapi.CallToolResult{
			Content: []mcpapi.CallToolResultContentElem{
				mcpapi.TextContent{ // legacy content - see https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content
					Type: "text",
					Text: string(unhealthyResourcesText),
				},
			},
			StructuredContent: unhealthyResourcesStructured,
			IsError:           mcpapi.BoolPtr(false),
		}
		if logger.Enabled(ctx, slog.LevelDebug) {
			logger.DebugContext(ctx, "returned 'tools/call' response", "content", result)
		}
		return result, nil
	}
}

func getUnhealthyResources(ctx context.Context, _ *slog.Logger, cl HTTPClient, name string) (*UnhealthyResources, error) {
	resp, err := cl.GetWithContext(ctx, fmt.Sprintf("api/v1/applications?name=%s", name)) // no heading `/` in the path
	if err != nil {
		return nil, fmt.Errorf("failed to get name '%s' from Argo CD: %w", name, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}
	defer resp.Body.Close()
	apps := &argocdv3.ApplicationList{}
	if err = json.Unmarshal(body, apps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal name list: %w", err)
	}
	if len(apps.Items) == 0 {
		return nil, fmt.Errorf("no name found with name %s", name)
	}
	app := apps.Items[0]
	// retain unhealthy resources from the name status
	unhealthyResources := &UnhealthyResources{
		Resources: []*argocdv3.ResourceResult{},
	}
	for _, resource := range app.Status.OperationState.SyncResult.Resources {
		if resource.Status != common.ResultCodeSynced {
			unhealthyResources.Resources = append(unhealthyResources.Resources, resource)
		}
	}
	return unhealthyResources, nil
}

// a wrapper, because `runtime.DefaultUnstructuredConverter.ToUnstructured`:
// - requires a pointer to a struct
// - does not support anonymous structs
type UnhealthyResources struct {
	Resources []*argocdv3.ResourceResult `json:"resources"`
}
