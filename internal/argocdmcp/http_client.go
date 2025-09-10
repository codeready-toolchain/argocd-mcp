package argocdmcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

type HTTPClient interface {
	GetWithContext(ctx context.Context, path string) (*http.Response, error)
}

type httpClient struct {
	insecure bool
	token    string
	url      string
}

func NewHTTPClient(url string, token string, insecure bool) HTTPClient {
	return httpClient{
		url:      url,
		token:    token,
		insecure: insecure,
	}
}

func (b httpClient) GetWithContext(ctx context.Context, path string) (*http.Response, error) {
	cl := http.DefaultClient
	// do not change the underlying transport of the default client if the insecure flag is not set (so we can test it with gock)
	if b.insecure {
		cl.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: b.insecure, //nolint:gosec
			},
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s", b.url, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))
	return cl.Do(req)
}
