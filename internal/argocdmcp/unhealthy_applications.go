package argocdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	mcpapi "github.com/xcoulon/converse-mcp/pkg/api"
	mcpserver "github.com/xcoulon/converse-mcp/pkg/server"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	argocdhealth "github.com/argoproj/gitops-engine/pkg/health"
)

var UnhealthyApplicationsTool = mcpapi.NewTool("unhealthyApplications").
	WithTitle("list the Unhealthy ('Degraded' and 'Progressing') Applications in Argo CD").
	WithDestructiveHint(false).
	WithReadOnlyHint(true)

func UnhealthyApplicationsToolHandle(logger *slog.Logger, cl HTTPClient) mcpserver.ToolHandleFunc {
	return func(ctx context.Context, _ mcpapi.CallToolRequestParams) (mcpapi.CallToolResult, error) {
		apps, err := listApplications(ctx, logger, cl)
		if err != nil {
			return mcpapi.CallToolResult{}, err
		}
		unhealthyApps := append(apps[argocdhealth.HealthStatusDegraded], apps[argocdhealth.HealthStatusProgressing]...)
		result := mcpapi.CallToolResult{
			Content: []mcpapi.CallToolResultContentElem{
				mcpapi.TextContent{ // legacy content - see https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content
					Type: "text",
					Text: strings.Join(unhealthyApps, ", "),
				},
			},
			StructuredContent: map[string]any{
				"degraded":    apps[argocdhealth.HealthStatusDegraded],
				"progressing": apps[argocdhealth.HealthStatusProgressing],
			},
			IsError: mcpapi.BoolPtr(false),
		}
		if logger.Enabled(ctx, slog.LevelDebug) {
			logger.DebugContext(ctx, "returned 'tools/call' response", "content", result)
		}
		return result, nil
	}
}

// returns the name of the applications grouped by their health status
func listApplications(ctx context.Context, _ *slog.Logger, cl HTTPClient) (map[argocdhealth.HealthStatusCode][]string, error) {
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
