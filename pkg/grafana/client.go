package grafana

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

type clientBase struct {
	baseURL             *url.URL
	httpClient          *http.Client
	authorizationHeader string
	basicAuth           *url.Userinfo
}

func NewWithApiKey(baseURL *url.URL, apiKey string) (Interface, error) {
	return NewWithApiKeyAndClient(baseURL, http.DefaultClient, apiKey)
}

func NewWithApiKeyAndClient(baseURL *url.URL, client *http.Client, apiKey string) (Interface, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("an API key is required to authenticate against the Grafana API")
	}

	return GrafanaClient{
		dashboards: DashboardsClient{
			clientBase{
				baseURL:             baseURL,
				httpClient:          client,
				authorizationHeader: fmt.Sprintf("Bearer %s", apiKey),
			},
		},
	}, nil
}

func NewWithUserCredentials(baseURL *url.URL, username, password string) (Interface, error) {
	return NewWithUserCredentialsAndClient(baseURL, http.DefaultClient, username, password)
}

func NewWithUserCredentialsAndClient(baseURL *url.URL, client *http.Client, username, password string) (Interface, error) {
	if username == "" {
		return nil, fmt.Errorf("a username is required to authenticate against the Grafana API")
	} else if password == "" {
		return nil, fmt.Errorf("a password is required to authenticate against the Grafana API")
	}

	return GrafanaClient{
		dashboards: DashboardsClient{
			clientBase{
				baseURL:    baseURL,
				httpClient: client,
				basicAuth:  url.UserPassword(username, password),
			},
		},
	}, nil
}

func (c *clientBase) newPostRequest(uri string, body []byte) (*http.Request, error) {
	return c.newRequest(http.MethodPost, uri, bytes.NewBuffer(body))
}

func (c *clientBase) newDeleteRequest(uri string) (*http.Request, error) {
	return c.newRequest(http.MethodDelete, uri, nil)
}

func (c *clientBase) newRequest(method, uri string, body io.Reader) (*http.Request, error) {
	url := c.baseURL
	url.Path = path.Join(url.Path, uri)
	if c.basicAuth != nil {
		url.User = c.basicAuth
	}

	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request : %v", err)
	}

	if c.authorizationHeader != "" {
		req.Header.Set("Authorization", c.authorizationHeader)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
