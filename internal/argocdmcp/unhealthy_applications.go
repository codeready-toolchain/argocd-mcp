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
	InputSchema:  UnhealthyApplicationsInputSchema,
	OutputSchema: UnhealthyApplicationsOutputSchema,
}

type UnhealthyApplicationsInput struct {
}

var UnhealthyApplicationsInputSchema, _ = jsonschema.For[UnhealthyApplicationsInput](&jsonschema.ForOptions{})

type UnhealthyApplicationsOutput struct {
	Degraded    []string `json:"degraded,omitempty"`
	Progressing []string `json:"progressing,omitempty"`
	OutOfSync   []string `json:"outOfSync,omitempty"`
}

var UnhealthyApplicationsOutputSchema, _ = jsonschema.For[UnhealthyApplicationsOutput](&jsonschema.ForOptions{})

func UnhealthyApplicationsToolHandle(logger *slog.Logger, cl *ArgoCDClient) mcp.ToolHandlerFor[UnhealthyApplicationsInput, UnhealthyApplicationsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ UnhealthyApplicationsInput) (*mcp.CallToolResult, UnhealthyApplicationsOutput, error) {
		apps, err := listUnhealthyApplications(ctx, logger, cl)
		if err != nil {
			return nil, UnhealthyApplicationsOutput{}, err
		}
		return nil, UnhealthyApplicationsOutput{
			Degraded:    apps[string(argocdhealth.HealthStatusDegraded)],
			Progressing: apps[string(argocdhealth.HealthStatusProgressing)],
			OutOfSync:   apps[string(argocdv3.SyncStatusCodeOutOfSync)],
		}, nil
	}
}

// returns the name of the applications grouped by their health status
func listUnhealthyApplications(ctx context.Context, logger *slog.Logger, cl *ArgoCDClient) (map[string][]string, error) {
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
	unhealthyApps := map[string][]string{}
	for _, app := range apps.Items {
		switch app.Status.Health.Status {
		case argocdhealth.HealthStatusDegraded, argocdhealth.HealthStatusProgressing:
			addApplicationToUnhealthyApps(unhealthyApps, string(app.Status.Health.Status), app.Name)
		case argocdhealth.HealthStatusHealthy:
			if app.Status.Sync.Status == argocdv3.SyncStatusCodeOutOfSync {
				addApplicationToUnhealthyApps(unhealthyApps, string(argocdv3.SyncStatusCodeOutOfSync), app.Name)
			}
		default:
			// skip healthy/synced apps
		}
	}

	if logger.Enabled(ctx, slog.LevelDebug) {
		unhealthyAppsStr, err := json.Marshal(unhealthyApps)
		if err != nil {
			logger.Error("failed to convert unhealthy resources to text", "error", err.Error())
		}
		logger.DebugContext(ctx, "returned 'tools/call' response", "tool", "unhealthyApplications", "result", string(unhealthyAppsStr))
	}
	return unhealthyApps, nil
}

func addApplicationToUnhealthyApps(unhealthyApps map[string][]string, status, name string) {
	if unhealthyApps[status] == nil {
		unhealthyApps[status] = []string{}
	}
	unhealthyApps[status] = append(unhealthyApps[status], name)
}
