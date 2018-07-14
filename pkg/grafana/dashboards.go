package grafana

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

const dashboardsPath = "api/dashboards/db"
const importPath = "api/dashboards/import"
const searchPath = "api/search"

type DashboardsClient struct {
	clientBase
}

func (c DashboardsClient) Import(ctx context.Context, dashboard *Dashboard) error {
	data, err := dashboard.data.marshalJSON()
	if err != nil {
		return fmt.Errorf("could not marshal dashboard: %v", err)
	}

	req, err := c.newPostRequest(ctx, importPath, data)
	if err != nil {
		return fmt.Errorf("could create import request : %v", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not import dashboard: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody := readErrorBodyAsString(res)
		return fmt.Errorf("bad response '%v'. could not import dashboard: %v", res.Status, errBody)
	}

	return nil
}

func (c DashboardsClient) ImportAndOverwrite(ctx context.Context, dashboard *Dashboard) error {
	dashboard.data.set("overwrite", true)
	return c.Import(ctx, dashboard)
}

func (c DashboardsClient) Delete(ctx context.Context, slug string) error {
	uri := path.Join(dashboardsPath, slug)
	req, err := c.newDeleteRequest(ctx, uri)
	if err != nil {
		return fmt.Errorf("could not create delete request for dashboard %s: %v", slug, err)
	}

	res, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("could not delete dashboard %s: %v", slug, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody := readErrorBodyAsString(res)
		return fmt.Errorf("bad response '%v'. could not delete dashboard %s: %v", res.Status, slug, errBody)
	}

	return nil
}

func (c DashboardsClient) Search(ctx context.Context, query DashboardSearchQuery) ([]*DashboardResult, error) {
	req, err := c.newGetRequest(ctx, searchPath, url.Values{
		"tag": query.Tags,
	})

	if err != nil {
		return nil, fmt.Errorf("could not build dashboards search request with query %v: %v", query, err)
	}
	res, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("could not search for dashboards with query %v: %v", query, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errBody := readErrorBodyAsString(res)
		return nil, fmt.Errorf("bad response '%v'. could not search for dashboards with query %v : %v", res.Status, query, errBody)
	}

	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, fmt.Errorf("could not read search results of dashboards search with query %v: %v", query, err)
	}

	return newDashboardSearchResults(bodyBytes)
}

func readErrorBodyAsString(res *http.Response) string {
	if res != nil && res.Body != nil {
		defer res.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		return string(bodyBytes)
	}
	return ""
}
