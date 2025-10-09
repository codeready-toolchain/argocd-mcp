package argocd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

type Client interface {
	GetWithContext(ctx context.Context, path string) (*http.Response, error)
}

type client struct {
	*http.Client
	host  string
	token string
}

func NewClient(host string, token string, insecure bool) Client {
	cl := http.DefaultClient
	if insecure {
		cl.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure, //nolint:gosec
			},
		}
	}
	return &client{
		Client: cl,
		host:   host,
		token:  token,
	}
}

func (c *client) GetWithContext(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s", c.host, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	return c.Do(req)
}
