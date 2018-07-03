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
	baseURL    *url.URL
	httpClient *http.Client
	auth       authInterface
}

func NewWithApiKey(baseURL *url.URL, apiKey string) (Interface, error) {
	return NewWithApiKeyAndClient(baseURL, http.DefaultClient, apiKey)
}

func NewWithApiKeyAndClient(baseURL *url.URL, client *http.Client, apiKey string) (Interface, error) {
	auth, err := newApiKeyAuth(apiKey)
	if err != nil {
		return nil, err
	}

	return newClientSet(baseURL, client, auth), nil
}

func NewWithUserCredentials(baseURL *url.URL, username, password string) (Interface, error) {
	return NewWithUserCredentialsAndClient(baseURL, http.DefaultClient, username, password)
}

func NewWithUserCredentialsAndClient(baseURL *url.URL, client *http.Client, username, password string) (Interface, error) {
	auth, err := newBasicAuth(username, password)
	if err != nil {
		return nil, err
	}

	return newClientSet(baseURL, client, auth), nil
}

func newClientSet(baseURL *url.URL, delegate *http.Client, auth authInterface) Interface {
	return GrafanaClient{
		dashboards: DashboardsClient{
			clientBase{
				baseURL,
				delegate,
				auth,
			},
		},
	}

}

func (c *clientBase) newGetRequest(uri string, params url.Values) (*http.Request, error) {
	return c.newRequest(http.MethodGet, uri, params, nil)
}

func (c *clientBase) newPostRequest(uri string, body []byte) (*http.Request, error) {
	return c.newRequest(http.MethodPost, uri, nil, bytes.NewBuffer(body))
}

func (c *clientBase) newDeleteRequest(uri string) (*http.Request, error) {
	return c.newRequest(http.MethodDelete, uri, nil, nil)
}

func (c *clientBase) newRequest(method, uri string, params url.Values, body io.Reader) (*http.Request, error) {
	url := c.baseURL
	url.Path = path.Join(url.Path, uri)
	if params != nil {
		url.RawQuery = params.Encode()
	}
	url = c.auth.authenticateUrl(url)

	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request : %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	return c.auth.authenticateRequest(req), nil
}

/*

.writeTimeout(5, SECONDS)
.readTimeout(5, SECONDS)
.connectTimeout(1, SECONDS)

   public String slug(String title) throws GrafanaException, IOException {
       return searchDashboard(title).get("uri").asText().substring(3);
   }


    public JsonNode searchDashboard(String title) throws GrafanaException, IOException {
        Response<List<JsonNode>> response = service.searchDashboard(title).execute();

        if (response.isSuccessful()) {
            if (response.body().size() == 1) {
                return response.body().get(0);
            } else {
                throw new DashboardDoesNotExistException(
                        format(
                                "Expected to find 1 dashboard with title %s, found %s",
                                title, response.body().size())
                );
			}



	...}}

*/
