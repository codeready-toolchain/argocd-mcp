package argocdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	argocdhealth "github.com/argoproj/gitops-engine/pkg/health"
)

var UnhealthyApplicationsTool = &mcp.Tool{
	Name:         "unhealthyApplications",
	Description:  "list the unhealthy ('degraded' and 'progressing') Applications in Argo CD",
	OutputSchema: UnhealthyApplicationsOutputSchema,
}

var UnhealthyApplicationsOutputSchema, _ = jsonschema.For[UnhealthyApplicationsOutput](&jsonschema.ForOptions{})

type UnhealthyApplicationsInput struct {
}

type UnhealthyApplicationsOutput struct {
	Degraded    []string `json:"degraded,omitempty"`
	Progressing []string `json:"progressing,omitempty"`
}

func UnhealthyApplicationsToolHandle(logger *slog.Logger, cl *ArgoCDClient) mcp.ToolHandlerFor[UnhealthyApplicationsInput, UnhealthyApplicationsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ UnhealthyApplicationsInput) (*mcp.CallToolResult, UnhealthyApplicationsOutput, error) {
		apps, err := listApplications(ctx, logger, cl)
		if err != nil {
			return nil, UnhealthyApplicationsOutput{}, err
		}
		return nil, UnhealthyApplicationsOutput{
			Degraded:    apps[argocdhealth.HealthStatusDegraded],
			Progressing: apps[argocdhealth.HealthStatusProgressing],
		}, nil
	}
}

// returns the name of the applications grouped by their health status
func listApplications(ctx context.Context, _ *slog.Logger, cl *ArgoCDClient) (map[argocdhealth.HealthStatusCode][]string, error) {
	resp, err := cl.GetWithContext(ctx, "api/v1/applications")
	if err != nil {
		return nil, err
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
	result := map[argocdhealth.HealthStatusCode][]string{}
	for _, app := range apps.Items {
		if result[app.Status.Health.Status] == nil {
			result[app.Status.Health.Status] = make([]string, 0, len(apps.Items))
		}
		result[app.Status.Health.Status] = append(result[app.Status.Health.Status], app.Name)
	}
	return result, nil
}
