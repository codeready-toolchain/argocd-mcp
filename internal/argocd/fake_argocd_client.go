package argocd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	testresources "github.com/codeready-toolchain/argocd-mcp/test/resources"
)

type FakeArgoCDClient struct {
}

func (c *FakeArgoCDClient) GetWithContext(_ context.Context, path string) (*http.Response, error) {
	switch path {
	case "api/v1/applications":
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testresources.ApplicationsStr)),
		}, nil
	case "api/v1/applications?name=example":
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testresources.ExampleApplicationStr)),
		}, nil
	case "api/v1/applications?name=example-error":
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	return nil, fmt.Errorf("not implemented: %s", path)
}
