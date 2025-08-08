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

var UnhealthyResourcesTool = mcpapi.Tool{
	Name:        "unhealthyResources",
	Description: mcpapi.ToStringPtr("get unhealthy resources of an Argo CD Application"),
	InputSchema: mcpapi.ToolInputSchema{
		Type: "object",
		Properties: map[string]map[string]any{
			"name": {
				"type":        "string",
				"description": mcpapi.ToStringPtr("the name of the Argo CD Application to get details of"),
			},
		},
		Required: []string{"name"},
	},
	Annotations: &mcpapi.ToolAnnotations{
		Title:           mcpapi.ToStringPtr("get unhealthy resources of an Argo CD Application"),
		DestructiveHint: mcpapi.ToBoolPtr(false),
		ReadOnlyHint:    mcpapi.ToBoolPtr(true),
	},
}

func UnhealthyResourcesHandle(cl HTTPClient) mcpserver.ToolHandleFunc {
	return func(ctx context.Context, logger *slog.Logger, params mcpapi.CallToolRequestParams) (any, error) {
		name, ok := params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name is not a string")
		}
		resp, err := cl.GetWithContext(ctx, fmt.Sprintf("api/v1/applications?name=%s", name)) // no heading `/` in the path
		if err != nil {
			return nil, fmt.Errorf("failed to get application '%s' from Argo CD: %w", name, err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
		}
		defer resp.Body.Close()
		apps := &argocdv3.ApplicationList{}
		if err = json.Unmarshal(body, apps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal application list: %w", err)
		}
		if len(apps.Items) == 0 {
			return nil, fmt.Errorf("no application found with name %s", name)
		}
		app := apps.Items[0]
		// retain unhealthy resources from the application status
		unhealthyResources := &UnhealthyResources{
			Resources: []*argocdv3.ResourceResult{},
		}
		for _, resource := range app.Status.OperationState.SyncResult.Resources {
			if resource.Status != common.ResultCodeSynced {
				unhealthyResources.Resources = append(unhealthyResources.Resources, resource)
			}
		}
		unhealthyResourcesText, err := json.Marshal(unhealthyResources)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unhealthy resources to text: %w", err)
		}
		unhealthyResourcesJSON, err := runtime.DefaultUnstructuredConverter.ToUnstructured(unhealthyResources)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unhealthy resources to unstructured content: %w", err)
		}
		result := &mcpapi.CallToolResult{
			Content: []mcpapi.CallToolResultContentElem{
				mcpapi.TextContent{ // legacy content - see https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content
					Type: "text",
					Text: string(unhealthyResourcesText),
				},
			},
			StructuredContent: unhealthyResourcesJSON,
			IsError:           mcpapi.ToBoolPtr(false),
		}
		if logger.Enabled(ctx, slog.LevelDebug) {
			logger.DebugContext(ctx, "returned 'tools/call' response", "content", result)
		}
		return result, nil
	}
}

// a wrapper, because `runtime.DefaultUnstructuredConverter.ToUnstructured`:
// - requires a pointer to a struct
// - does not support anonymous structs
type UnhealthyResources struct {
	Resources []*argocdv3.ResourceResult `json:"resources"`
}
