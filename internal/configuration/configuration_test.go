package configuration_test

import (
	"os"
	"testing"

	"github.com/xcoulon/argocd-mcp/internal/configuration"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfiguration(t *testing.T) {

	t.Run("using plain values", func(t *testing.T) {
		// given
		f := flag.NewFlagSet("test", flag.ContinueOnError)

		// when
		cfg, err := configuration.NewFromFlagSet(f, []string{"--argocd-url", "https://argocd-server", "--argocd-token", "secure-token", "--insecure", "true"})

		// then
		require.NoError(t, err)
		assert.Equal(t, "https://argocd-server", cfg.URL)
		assert.Equal(t, "secure-token", cfg.Token)
		assert.True(t, cfg.Insecure)
	})

	t.Run("using env var expansion", func(t *testing.T) {
		// given
		os.Setenv("ARGOCD_URL", "https://argocd-server")
		defer os.Unsetenv("ARGOCD_URL")
		os.Setenv("ARGOCD_TOKEN", "secure-token")
		defer os.Unsetenv("ARGOCD_TOKEN")
		os.Setenv("ARGOCD_INSECURE", "true")
		defer os.Unsetenv("ARGOCD_INSECURE")
		f := flag.NewFlagSet("test", flag.ContinueOnError)

		// when
		cfg, err := configuration.NewFromFlagSet(f, []string{"--argocd-url", "$ARGOCD_URL", "--argocd-token", "$ARGOCD_TOKEN", "--insecure", "$ARGOCD_INSECURE"})

		// then
		require.NoError(t, err)
		assert.Equal(t, "https://argocd-server", cfg.URL)
		assert.Equal(t, "secure-token", cfg.Token)
		assert.True(t, cfg.Insecure)
	})
}
