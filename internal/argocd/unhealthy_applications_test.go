package argocd

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUnhealthyApplications(t *testing.T) {
	// given
	cl := &FakeArgoCDClient{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// when
	unhealthyApps, err := listUnhealthyApplications(context.Background(), logger, cl)

	// then
	require.NoError(t, err)
	assert.Equal(t, UnhealthyApplications{
		Degraded:    []string{"a-degraded-application", "another-degraded-application"},
		Progressing: []string{"a-progressing-application", "another-progressing-application"},
		OutOfSync:   []string{"an-out-of-sync-application", "another-out-of-sync-application"},
		Missing:     nil, // TODO: add missing applications
		Unknown:     nil, // TODO: add unknown applications
		Suspended:   nil, // TODO: add suspended applications
	}, unhealthyApps)
}
