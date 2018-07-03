package grafana

import (
	"fmt"
	"net/http"
	"net/url"
)

type authInterface interface {
	authenticateUrl(url *url.URL) *url.URL
	authenticateRequest(req *http.Request) *http.Request
}

type basicAuth struct {
	userInfo *url.Userinfo
}

func newBasicAuth(username string, password string) (*basicAuth, error) {
	if username == "" {
		return nil, fmt.Errorf("a username is required to authenticate against the Grafana API")
	} else if password == "" {
		return nil, fmt.Errorf("a password is required to authenticate against the Grafana API")
	}

	return &basicAuth{
		userInfo: url.UserPassword(username, password),
	}, nil
}

func (a *basicAuth) authenticateUrl(url *url.URL) *url.URL {
	url.User = a.userInfo
	return url
}
func (a *basicAuth) authenticateRequest(req *http.Request) *http.Request { return req }

type apiKeyAuth struct {
	authorizationHeader string
}

func newApiKeyAuth(apiKey string) (*apiKeyAuth, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("an API key is required to authenticate against the Grafana API")
	}

	return &apiKeyAuth{
		authorizationHeader: fmt.Sprintf("Bearer %s", apiKey),
	}, nil
}

func (a *apiKeyAuth) authenticateRequest(req *http.Request) *http.Request {
	req.Header.Set("Authorization", a.authorizationHeader)
	return req
}
func (a *apiKeyAuth) authenticateUrl(url *url.URL) *url.URL { return url }
