package grafana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

const dashboardsPath = "api/dashboards/db"
const searchPath = "api/search"
const importPath = "api/dashboards/import"
const healthPath = "api/health"

type DashboardsClient struct {
	clientBase
}

func (c DashboardsClient) Import(dashboard Dashboard) error {
	data, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("could not marshal %v: %v", dashboard, err)
	}

	req, err := c.newPostRequest(importPath, data)
	if err != nil {
		return fmt.Errorf("error while importing dashboard: could create request POST %s with body %v : %v", importPath, dashboard, err)
	}

	res, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("error while importing dashboard %v: %v", dashboard, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("error while importing dashboard (%s) %v: %v", res.Status, dashboard, err)
	}

	return nil
}
func (c DashboardsClient) Delete(slug string) error {
	uri := path.Join(dashboardsPath, slug)
	req, err := c.newDeleteRequest(uri)
	if err != nil {
		return fmt.Errorf("error while deleting dashboard %s: could create request %s %s : %v", slug, http.MethodDelete, uri, err)
	}

	res, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("error while deleting dashboard %s: %v", slug, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("error while deleting dashboard (%s) %s: %v", res.Status, slug, err)
	}

	return nil
}

func (c DashboardsClient) Search(title string) ([]Dashboard, error) {
	uri := searchPath

	params := url.Values{}
	params.Set("title", title)

	req, err := c.newGetRequest(uri, params)
	if err != nil {
		return nil, fmt.Errorf("error while searching for dashboards with %v: %v", params, err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while searching for dashboards with %v: %v", params, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("error while searching for dashboards with %v (%s): %v", params, res.Status, err)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error while searching for dashboards with %v. could not read the response's body: %v", params, err)
	}

	var data []Dashboard
	err = json.Unmarshal(bytes, data)
	if err != nil {
		return nil, fmt.Errorf("error while searching for dashboards with %v. could not read the results of the search: %v", params, err)
	}

	return data, nil
}
