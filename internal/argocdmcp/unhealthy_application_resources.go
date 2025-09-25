package argocdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var UnhealthyResourcesPrompt = &mcp.Prompt{
	Name:        "argocd-unhealthy-application-resources",
	Description: "The unhealthy resources of the Argo CD Application prompt",
	Arguments: []*mcp.PromptArgument{
		{
			Name:        "name",
			Description: "the name of the application to get details of",
			Required:    true,
		},
	},
}

func UnhealthyApplicationResourcesPromptHandle(logger *slog.Logger, cl *ArgoCDClient) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		app, ok := req.Params.Arguments["name"]
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
		result := &mcp.GetPromptResult{
			Description: "The unhealthy resources of the Argo CD Application prompt",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
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

var UnhealthyApplicationResourcesTool = &mcp.Tool{
	Name:        "unhealthyApplicationResources",
	Description: "list unhealthy resources of a given Argo CD Application",
	InputSchema: &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"name": {
				Type:        "string",
				Description: "the name of the Argo CD Application to get details of",
			},
		},
		Required: []string{"name"},
	},
}

type UnhealthyApplicationResourcesInput struct {
	Name string `json:"name"`
}

type UnhealthyApplicationResourcesOutput struct {
	Resources []argocdv3.ResourceStatus `json:"resources"`
}

func UnhealthyApplicationResourcesToolHandle(logger *slog.Logger, cl *ArgoCDClient) mcp.ToolHandlerFor[UnhealthyApplicationResourcesInput, UnhealthyApplicationResourcesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in UnhealthyApplicationResourcesInput) (*mcp.CallToolResult, UnhealthyApplicationResourcesOutput, error) {
		unhealthyResources, err := getUnhealthyResources(ctx, logger, cl, in.Name)
		if err != nil {
			return nil, UnhealthyApplicationResourcesOutput{}, err
		}
		// unhealthyResourcesText, err := json.Marshal(unhealthyResources)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to convert unhealthy resources to 'text' content: %w", err)
		// }
		// unhealthyResourcesStructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(unhealthyResources)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to convert unhealthy resources to 'structured' content: %w", err)
		// }
		// result := &mcp.CallToolResult{
		// 	Content: []mcp.Content{
		// 		&mcp.TextContent{ // legacy content - see https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content
		// 			Text: string(unhealthyResourcesText),
		// 		},
		// 	},
		// 	StructuredContent: unhealthyResourcesStructured,
		// 	IsError:           false,
		// }
		// if logger.Enabled(ctx, slog.LevelDebug) {
		// 	logger.DebugContext(ctx, "returned 'tools/call' response", "content", result)
		// }
		return nil, UnhealthyApplicationResourcesOutput{
			Resources: unhealthyResources.Resources,
		}, nil
	}
}

func getUnhealthyResources(ctx context.Context, _ *slog.Logger, cl *ArgoCDClient, name string) (*UnhealthyResources, error) {
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
		Resources: []argocdv3.ResourceStatus{},
	}
	for _, resource := range app.Status.Resources {
		if resource.Health != nil && resource.Health.Status != health.HealthStatusHealthy {
			unhealthyResources.Resources = append(unhealthyResources.Resources, resource)
		}
	}
	return unhealthyResources, nil
}

// a wrapper, because `runtime.DefaultUnstructuredConverter.ToUnstructured`:
// - requires a pointer to a struct
// - does not support anonymous structs
type UnhealthyResources struct {
	Resources []argocdv3.ResourceStatus `json:"resources"`
}
