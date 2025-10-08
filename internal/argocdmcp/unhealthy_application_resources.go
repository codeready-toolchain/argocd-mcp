package argocdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

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
		unhealthyResources, err := listUnhealthyApplicationResources(ctx, logger, cl, app)
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
	OutputSchema: UnhealthyApplicationResourcesOutputSchema,
}

var UnhealthyApplicationResourcesOutputSchema, _ = jsonschema.For[UnhealthyApplicationResourcesOutput](&jsonschema.ForOptions{})

type UnhealthyApplicationResourcesInput struct {
	Name string `json:"name"`
}

type UnhealthyApplicationResourcesOutput struct {
	Resources []argocdv3.ResourceStatus `json:"resources"`
}

func UnhealthyApplicationResourcesToolHandle(logger *slog.Logger, cl *ArgoCDClient) mcp.ToolHandlerFor[UnhealthyApplicationResourcesInput, UnhealthyApplicationResourcesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in UnhealthyApplicationResourcesInput) (*mcp.CallToolResult, UnhealthyApplicationResourcesOutput, error) {
		unhealthyResources, err := listUnhealthyApplicationResources(ctx, logger, cl, in.Name)
		if err != nil {
			return nil, UnhealthyApplicationResourcesOutput{}, err
		}
		return nil, UnhealthyApplicationResourcesOutput{
			Resources: unhealthyResources.Resources,
		}, nil
	}
}

func listUnhealthyApplicationResources(ctx context.Context, logger *slog.Logger, cl *ArgoCDClient, name string) (*UnhealthyResources, error) {
	resp, err := cl.GetWithContext(ctx, fmt.Sprintf("api/v1/applications?name=%s", name)) // no heading `/` in the path
	if err != nil {
		return nil, fmt.Errorf("failed to get application '%s' from Argo CD: %w", name, err)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected Argo CD status %d for application '%s': %s", resp.StatusCode, name, string(body))
	}
	apps := &argocdv3.ApplicationList{}
	if err = json.Unmarshal(body, apps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal application list: %w", err)
	}
	if len(apps.Items) == 0 {
		return nil, fmt.Errorf("no application found with name %s", name)
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
	if logger.Enabled(ctx, slog.LevelDebug) {
		unhealthyResourcesStr, err := json.Marshal(unhealthyResources)
		if err != nil {
			logger.Error("failed to convert unhealthy resources to text", "error", err.Error())
		}
		logger.DebugContext(ctx, "returned 'tools/call' response", "tool", "unhealthyApplicationResources", "app", name, "result", string(unhealthyResourcesStr))
	}
	return unhealthyResources, nil
}

// a wrapper, because `runtime.DefaultUnstructuredConverter.ToUnstructured`:
// - requires a pointer to a struct
// - does not support anonymous structs
type UnhealthyResources struct {
	Resources []argocdv3.ResourceStatus `json:"resources"`
}
