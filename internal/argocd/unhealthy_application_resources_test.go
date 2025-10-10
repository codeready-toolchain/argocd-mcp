package argocd

import (
	"context"
	"log/slog"
	"os"
	"testing"

	argocdv3 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUnhealthyApplicationResources(t *testing.T) {

	t.Run("example", func(t *testing.T) {
		// given
		cl := &FakeArgoCDClient{}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// when
		unhealthyResources, err := listUnhealthyApplicationResources(context.Background(), logger, cl, "example")

		// then
		require.NoError(t, err)
		assert.Equal(t, UnhealthyResources{
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
				{
					Group:   "operator.tekton.dev",
					Version: "v1alpha1",
					Kind:    "TektonConfig",
					Name:    "config",
					Status:  "OutOfSync",
				},
			},
		}, unhealthyResources)
	})
}
