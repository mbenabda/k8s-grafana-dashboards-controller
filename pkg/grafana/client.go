package grafana

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

type grafanaClient struct {
	dashboards DashboardsInterface
}

func (c grafanaClient) Dashboards() DashboardsInterface {
	return c.dashboards
}

type clientBase struct {
	baseURL     *url.URL
	httpClient  *http.Client
	bearerToken string
	basicAuth   *url.Userinfo
}

func NewWithApiKey(baseURL *url.URL, apiKey string) (Interface, error) {
	return NewWithApiKeyAndClient(baseURL, http.DefaultClient, apiKey)
}

func NewWithApiKeyAndClient(baseURL *url.URL, client *http.Client, apiKey string) (Interface, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("an API key is required to authenticate against the Grafana API")
	}

	return grafanaClient{
		dashboards: dashboardsClient{
			clientBase{
				baseURL:     baseURL,
				httpClient:  client,
				bearerToken: apiKey,
			},
		},
	}, nil
}

func NewWithBasicAuth(baseURL *url.URL, username, password string) (Interface, error) {
	return NewWithBasicAuthAndClient(baseURL, http.DefaultClient, username, password)
}

func NewWithBasicAuthAndClient(baseURL *url.URL, client *http.Client, username, password string) (Interface, error) {
	if username == "" {
		return nil, fmt.Errorf("a username is required to authenticate against the Grafana API")
	}

	return grafanaClient{
		dashboards: dashboardsClient{
			clientBase{
				baseURL:    baseURL,
				httpClient: client,
				basicAuth:  url.UserPassword(username, password),
			},
		},
	}, nil
}

func (c *clientBase) newPostRequest(ctx context.Context, uri string, body []byte) (*http.Request, error) {
	return c.newRequest(ctx, http.MethodPost, uri, nil, bytes.NewBuffer(body))
}

func (c *clientBase) newDeleteRequest(ctx context.Context, uri string) (*http.Request, error) {
	return c.newRequest(ctx, http.MethodDelete, uri, nil, nil)
}

func (c *clientBase) newGetRequest(ctx context.Context, uri string, params url.Values) (*http.Request, error) {
	return c.newRequest(ctx, http.MethodGet, uri, params, nil)
}

func (c *clientBase) newRequest(ctx context.Context, method, uri string, params url.Values, body io.Reader) (*http.Request, error) {
	url, err := url.Parse(c.baseURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse url %v: %v", c.baseURL.String(), err)
	}

	url.Path = path.Join(url.Path, uri)
	if params != nil {
		url.RawQuery = params.Encode()
	}

	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request : %v", err)
	}

	c.applyAuth(req)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return req.WithContext(ctx), nil
}

func (c *clientBase) applyAuth(req *http.Request) {
	if c.bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
	} else if c.basicAuth != nil {
		p, _ := c.basicAuth.Password()
		req.SetBasicAuth(c.basicAuth.Username(), p)
	}
}
